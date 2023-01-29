import libav from 'libav.js/libav-3.9.5.1-webm.js'
import 'libav.js/libav-3.9.5.1-webm.wasm.wasm'
import 'libav.js/libav-3.9.5.1-webm.wasm.js'
import 'libav.js/libav-3.9.5.1-webm.simd.wasm'
import 'libav.js/libav-3.9.5.1-webm.simd.js'

import * as libavwebcodecs from 'libavjs-webcodecs-polyfill'

export async function CheckWebCodecSupport() {
    if (!('VideoEncoder' in window)) {
        console.log("WebCodec support missing. Loading polyfil")

        await new Promise((res, rej) => {
                window.LibAV = {base: "static/libav"};
                const scr = document.createElement("script");
                scr.src = libav;
                scr.onload = res;
                scr.onerror = rej;
                document.body.appendChild(scr);
            }
        );

        await libavwebcodecs.load({
            polyfill: true,
            libavOptions: {
                // SIMD is broken
                nosimd: true,
            }
        });

        return false;
    }

    return true;
}
