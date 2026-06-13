// AMT KVM client — a port of MeshCommander's Intel AMT RFB implementation.
//
// Intel AMT KVM is a custom RFB (VNC) variant, not standard VNC:
//   - default pixel format RGB565 (2 bytes/pixel, little-endian pixels)
//   - tiles are at most 64x64
//   - encodings: RAW (0), AMT RLE/LRE (16, raw-DEFLATE wrapped), Desktop Size (-223)
//   - RFB framing is big-endian; pixel data is little-endian
//   - security is skipped (None) because the redirection layer already did digest auth
//
// The Go sidecar performs the redirection handshake + auth and then pipes the
// raw RFB byte stream over the WebSocket; this class decodes it to a canvas and
// sends input back.
import { Inflate } from "fflate";
import { clientLog } from "./logger";

const SET_PIXEL_NONE = 2; // bytes per pixel (RGB565)

export type KvmStateHandler = (state: "init" | "running" | "error" | "closed", detail?: string) => void;

export class AmtKvmClient {
  private ctx: CanvasRenderingContext2D;
  private send: (b: Uint8Array) => void;
  private onState: KvmStateHandler;

  private acc: Uint8Array | null = null;
  private state = 0;
  private width = 0;
  private height = 0;
  private readonly bpp = SET_PIXEL_NONE;

  // Reusable tile bitmap (the "spare").
  private spare: ImageData | null = null;
  private sparew = 0;
  private spareh = 0;

  // Streaming raw-DEFLATE inflater for RLE-encoded tiles.
  private inflate!: Inflate;
  private inflateOut: Uint8Array[] = [];

  // Diagnostics (bounded so the log isn't flooded).
  private totalBytes = 0;
  private fbUpdates = 0;
  private loggedFirstData = false;

  constructor(
    canvas: HTMLCanvasElement,
    send: (b: Uint8Array) => void,
    onState: KvmStateHandler,
  ) {
    this.ctx = canvas.getContext("2d")!;
    this.send = send;
    this.onState = onState;
    this.resetInflate();
  }

  private resetInflate() {
    this.inflate = new Inflate();
    this.inflate.ondata = (chunk) => this.inflateOut.push(chunk);
  }

  /** Feed raw RFB bytes received from the device. */
  processData(data: ArrayBuffer | Uint8Array) {
    const incoming = data instanceof Uint8Array ? data : new Uint8Array(data);
    this.totalBytes += incoming.byteLength;
    if (!this.loggedFirstData) {
      this.loggedFirstData = true;
      void clientLog("info", `kvm: first data (${incoming.byteLength} bytes), state=${this.state}`, "kvm");
    }
    if (!this.acc) {
      this.acc = incoming;
    } else {
      const tmp = new Uint8Array(this.acc.byteLength + incoming.byteLength);
      tmp.set(this.acc, 0);
      tmp.set(incoming, this.acc.byteLength);
      this.acc = tmp;
    }
    this.pump();
  }

  private pump() {
    while (this.acc && this.acc.byteLength > 0) {
      const acc = this.acc;
      const view = new DataView(acc.buffer, acc.byteOffset, acc.byteLength);
      let cmdsize = 0;

      if (this.state === 0 && acc.byteLength >= 12) {
        // ProtocolVersion: "RFB 003.008\n"
        cmdsize = 12;
        this.state = 1;
        this.onState("init", "version");
        this.sendStr("RFB 003.008\n");
      } else if (this.state === 1 && acc.byteLength >= 1) {
        // Security types list: [count][types...]; reply 'None' (1).
        cmdsize = acc[0] + 1;
        this.sendBytes([1]);
        this.state = 2;
      } else if (this.state === 2 && acc.byteLength >= 4) {
        // SecurityResult (big-endian u32, 0 = OK). Then ClientInit (shared=1).
        cmdsize = 4;
        if (view.getUint32(0) !== 0) return this.fail("security rejected");
        this.sendBytes([1]);
        this.state = 3;
      } else if (this.state === 3 && acc.byteLength >= 24) {
        // ServerInit: width/height (BE u16) + pixel format + name.
        const namelen = view.getUint32(20);
        if (acc.byteLength < 24 + namelen) return;
        cmdsize = 24 + namelen;
        this.width = view.getUint16(0);
        this.height = view.getUint16(2);
        this.ctx.canvas.width = this.width;
        this.ctx.canvas.height = this.height;

        // Log the server's pixel format so we can confirm the RGB565 assumption.
        void clientLog(
          "info",
          `kvm: ServerInit ${this.width}x${this.height} bpp=${acc[4]} depth=${acc[5]} ` +
            `bigEndian=${acc[6]} trueColor=${acc[7]} ` +
            `max(r=${view.getUint16(8)},g=${view.getUint16(10)},b=${view.getUint16(12)}) ` +
            `shift(r=${acc[14]},g=${acc[15]},b=${acc[16]})`,
          "kvm",
        );

        // SetEncodings: RLE(16), RAW(0), 1092 (Intel), DesktopSize(-223).
        const enc = [16, 0, 1092, -223];
        const msg: number[] = [2, 0, (enc.length >> 8) & 0xff, enc.length & 0xff];
        for (const e of enc) msg.push((e >> 24) & 0xff, (e >> 16) & 0xff, (e >> 8) & 0xff, e & 0xff);
        this.sendBytes(msg);

        this.state = 4;
        this.onState("running");
        void clientLog("info", "kvm: handshake complete, requesting framebuffer", "kvm");
        this.sendRefresh();
      } else if (this.state === 4) {
        // Server message type.
        switch (acc[0]) {
          case 0: { // FramebufferUpdate
            if (acc.byteLength < 4) return;
            const rects = view.getUint16(2);
            this.state = 100 + rects; // pending rect count
            cmdsize = 4;
            if (this.fbUpdates < 3) {
              this.fbUpdates++;
              void clientLog("info", `kvm: FramebufferUpdate #${this.fbUpdates}, ${rects} rect(s)`, "kvm");
            }
            // A 0-rect update would otherwise wedge the state machine; recover
            // by re-requesting and returning to the message-dispatch state.
            if (rects === 0) {
              this.state = 4;
              this.sendRefresh();
            }
            break;
          }
          case 2: // Bell
            cmdsize = 1;
            break;
          case 3: { // ServerCutText
            if (acc.byteLength < 8) return;
            const len = view.getUint32(4) + 8;
            if (acc.byteLength < len) return;
            cmdsize = len;
            break;
          }
          default:
            return this.fail(`unexpected RFB message ${acc[0]}`);
        }
      } else if (this.state > 100 && acc.byteLength >= 12) {
        cmdsize = this.decodeRect(acc, view);
        if (cmdsize === 0) return; // need more bytes
      } else {
        return; // need more bytes
      }

      if (cmdsize === 0) return;
      this.acc = cmdsize === acc.byteLength ? null : acc.subarray(cmdsize);
    }
  }

  /** Decode one framebuffer rect. Returns bytes consumed, or 0 if incomplete. */
  private decodeRect(acc: Uint8Array, view: DataView): number {
    const x = view.getUint16(0);
    const y = view.getUint16(2);
    const width = view.getUint16(4);
    const height = view.getUint16(6);
    const s = width * height;
    const encoding = view.getUint32(8);

    if (encoding < 17) {
      if (width < 1 || width > 64 || height < 1 || height > 64) {
        this.fail(`invalid tile size ${width}x${height}`);
        return 0;
      }
      this.ensureSpare(width, height);
    }

    let cmdsize = 0;
    if (encoding === 0xffffff21) {
      // Desktop Size pseudo-encoding.
      this.width = width;
      this.height = height;
      this.ctx.canvas.width = width;
      this.ctx.canvas.height = height;
      this.fbUpdateRequest(false, 0, 0, width, height);
      cmdsize = 12;
    } else if (encoding === 0) {
      // RAW: width*height pixels, RGB565 little-endian.
      const cs = 12 + s * this.bpp;
      if (acc.byteLength < cs) return 0;
      let ptr = 12;
      for (let i = 0; i < s; i++) {
        this.setPixel16(view.getUint16(ptr, true), i);
        ptr += 2;
      }
      this.putImage(x, y);
      cmdsize = cs;
    } else if (encoding === 16) {
      // AMT RLE: 4-byte length then raw-DEFLATE stream of LRE data.
      if (acc.byteLength < 16) return 0;
      const datalen = view.getUint32(12);
      if (acc.byteLength < 16 + datalen) return 0;
      const p = 16;
      // Fast path: an uncompressed DEFLATE "stored" block (the default, since
      // AMT compression is off unless explicitly enabled) — decode the LRE
      // payload directly, skipping the 5-byte stored-block header. This mirrors
      // MeshCommander and avoids the inflater entirely.
      if (datalen > 5 && acc[p] === 0 && (acc[p + 1] | (acc[p + 2] << 8)) === datalen - 5) {
        this.decodeLRE(acc, p + 5, x, y, width, height, s);
      } else {
        const lre = this.inflateBlock(acc.subarray(p, p + datalen));
        this.decodeLRE(lre, 0, x, y, width, height, s);
      }
      cmdsize = 16 + datalen;
    } else {
      this.fail(`unsupported encoding ${encoding}`);
      return 0;
    }

    // One rect consumed; drop pending count and refresh when the frame is done.
    if (--this.state === 100) {
      this.state = 4;
      this.sendRefresh();
    }
    return cmdsize;
  }

  /** Inflate one RLE block through the persistent raw-DEFLATE stream. */
  private inflateBlock(block: Uint8Array): Uint8Array {
    this.inflateOut = [];
    try {
      this.inflate.push(block, false);
    } catch {
      // Fall back to treating the block as an uncompressed stored deflate body.
    }
    if (this.inflateOut.length === 1) return this.inflateOut[0];
    let total = 0;
    for (const c of this.inflateOut) total += c.byteLength;
    const out = new Uint8Array(total);
    let off = 0;
    for (const c of this.inflateOut) {
      out.set(c, off);
      off += c.byteLength;
    }
    return out;
  }

  /** Decode an AMT LRE (run-length/palette) tile payload into the spare bitmap. */
  private decodeLRE(data: Uint8Array, ptr: number, x: number, y: number, width: number, height: number, s: number) {
    if (data.byteLength === 0) return;
    const sub = data[ptr++];
    if (sub === 0) {
      // RAW within LRE.
      for (let i = 0; i < s; i++) {
        this.setPixel16(data[ptr] + (data[ptr + 1] << 8), i);
        ptr += 2;
      }
      this.putImage(x, y);
    } else if (sub === 1) {
      // Solid color tile.
      const v = data[ptr] + (data[ptr + 1] << 8);
      this.ctx.fillStyle = rgb565(v);
      this.ctx.fillRect(x, y, width, height);
    } else if (sub > 1 && sub < 17) {
      // Packed palette.
      const palette: number[] = [];
      for (let i = 0; i < sub; i++) {
        palette[i] = data[ptr] + (data[ptr + 1] << 8);
        ptr += 2;
      }
      let br = 4, bm = 15;
      if (sub === 2) { br = 1; bm = 1; } else if (sub <= 4) { br = 2; bm = 3; }
      let rle = 0;
      while (rle < s && ptr < data.byteLength) {
        const v = data[ptr++];
        for (let i = 8 - br; i >= 0; i -= br) this.setPixel16(palette[(v >> i) & bm], rle++);
      }
      this.putImage(x, y);
    } else if (sub === 128) {
      // RLE run-length tile.
      let rle = 0;
      while (rle < s && ptr < data.byteLength) {
        const v = data[ptr++] + (data[ptr++] << 8);
        let runlength = 1, d: number;
        do { d = data[ptr++]; runlength += d; } while (d === 255);
        this.setPixel16Run(v, rle, runlength);
        rle += runlength;
      }
      this.putImage(x, y);
    } else if (sub > 129) {
      // Palette RLE.
      const palette: number[] = [];
      for (let i = 0; i < sub - 128; i++) {
        palette[i] = data[ptr] + (data[ptr + 1] << 8);
        ptr += 2;
      }
      let rle = 0;
      while (rle < s && ptr < data.byteLength) {
        let runlength = 1;
        const index = data[ptr++];
        const v = palette[index % 128];
        if (index > 127) {
          let d: number;
          do { d = data[ptr++]; runlength += d; } while (d === 255);
        }
        this.setPixel16Run(v, rle, runlength);
        rle += runlength;
      }
      this.putImage(x, y);
    }
  }

  private ensureSpare(width: number, height: number) {
    if (this.sparew !== width || this.spareh !== height || !this.spare) {
      this.sparew = width;
      this.spareh = height;
      this.spare = this.ctx.createImageData(width, height);
      const d = this.spare.data;
      for (let i = 3; i < d.length; i += 4) d[i] = 0xff; // opaque alpha
    }
  }

  private setPixel16(v: number, p: number) {
    const pp = p << 2;
    const d = this.spare!.data;
    d[pp] = (v >> 8) & 248;
    d[pp + 1] = (v >> 3) & 252;
    d[pp + 2] = (v & 31) << 3;
  }

  private setPixel16Run(v: number, p: number, run: number) {
    const d = this.spare!.data;
    let pp = p << 2;
    const r = (v >> 8) & 248, g = (v >> 3) & 252, b = (v & 31) << 3;
    while (--run >= 0) {
      d[pp] = r;
      d[pp + 1] = g;
      d[pp + 2] = b;
      pp += 4;
    }
  }

  private putImage(x: number, y: number) {
    this.ctx.putImageData(this.spare!, x, y);
  }

  // --- outbound RFB messages ---

  private sendRefresh() {
    this.fbUpdateRequest(true, 0, 0, this.width, this.height);
  }

  private fbUpdateRequest(incremental: boolean, x: number, y: number, w: number, h: number) {
    this.sendBytes([3, incremental ? 1 : 0, (x >> 8) & 0xff, x & 0xff, (y >> 8) & 0xff, y & 0xff,
      (w >> 8) & 0xff, w & 0xff, (h >> 8) & 0xff, h & 0xff]);
  }

  /** KeyEvent (type 4): down/up + 4-byte big-endian keysym. */
  sendKey(keysym: number, down: boolean) {
    this.sendBytes([4, down ? 1 : 0, 0, 0,
      (keysym >> 24) & 0xff, (keysym >> 16) & 0xff, (keysym >> 8) & 0xff, keysym & 0xff]);
  }

  /** Ctrl+Alt+Del sequence. */
  sendCtrlAltDel() {
    const seq: [number, boolean][] = [
      [0xffe3, true], [0xffe9, true], [0xffff, true],
      [0xffff, false], [0xffe9, false], [0xffe3, false],
    ];
    for (const [k, d] of seq) this.sendKey(k, d);
  }

  /** PointerEvent (type 5): button mask + big-endian x,y. */
  sendPointer(x: number, y: number, mask: number) {
    this.sendBytes([5, mask & 0xff, (x >> 8) & 0xff, x & 0xff, (y >> 8) & 0xff, y & 0xff]);
  }

  private sendStr(s: string) {
    const b = new Uint8Array(s.length);
    for (let i = 0; i < s.length; i++) b[i] = s.charCodeAt(i);
    this.send(b);
  }

  private sendBytes(arr: number[]) {
    this.send(new Uint8Array(arr));
  }

  private fail(detail: string) {
    void clientLog("error", `kvm: ${detail} (state=${this.state}, bytes=${this.totalBytes})`, "kvm");
    this.onState("error", detail);
  }
}

function rgb565(v: number): string {
  return `rgb(${(v >> 8) & 248},${(v >> 3) & 252},${(v & 31) << 3})`;
}

// Maps a KeyboardEvent.code to the AMT/X11 keysym, mirroring MeshCommander's
// keyboard-layout-independent table. Returns null for unknown keys.
const KEYMAP: Record<string, number> = {
  Pause: 19, CapsLock: 20, Space: 32, Quote: 39, Minus: 45, NumpadMultiply: 42,
  NumpadAdd: 43, PrintScreen: 44, Comma: 44, NumpadSubtract: 45, NumpadDecimal: 46,
  Period: 46, Slash: 47, NumpadDivide: 47, Semicolon: 59, Equal: 61, OSLeft: 91,
  BracketLeft: 91, OSRight: 91, Backslash: 92, BracketRight: 93, ContextMenu: 93,
  Backquote: 96, NumLock: 144, ScrollLock: 145, Backspace: 0xff08, Tab: 0xff09,
  Enter: 0xff0d, NumpadEnter: 0xff0d, Escape: 0xff1b, Delete: 0xffff, Home: 0xff50,
  PageUp: 0xff55, PageDown: 0xff56, ArrowLeft: 0xff51, ArrowUp: 0xff52,
  ArrowRight: 0xff53, ArrowDown: 0xff54, End: 0xff57, Insert: 0xff63,
  F1: 0xffbe, F2: 0xffbf, F3: 0xffc0, F4: 0xffc1, F5: 0xffc2, F6: 0xffc3,
  F7: 0xffc4, F8: 0xffc5, F9: 0xffc6, F10: 0xffc7, F11: 0xffc8, F12: 0xffc9,
  ShiftLeft: 0xffe1, ShiftRight: 0xffe2, ControlLeft: 0xffe3, ControlRight: 0xffe4,
  AltLeft: 0xffe9, AltRight: 0xffea, MetaLeft: 0xffe7, MetaRight: 0xffe8,
};

export function amtKeyFromEvent(e: KeyboardEvent): number | null {
  const code = e.code;
  if (code.startsWith("Key") && code.length === 4) {
    return code.charCodeAt(3) + (e.shiftKey ? 0 : 32);
  }
  if (code.startsWith("Digit") && code.length === 6) return code.charCodeAt(5);
  if (code.startsWith("Numpad") && code.length === 7) return code.charCodeAt(6);
  return KEYMAP[code] ?? null;
}
