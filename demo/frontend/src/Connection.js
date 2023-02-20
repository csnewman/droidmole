import {CheckWebCodecSupport} from "./CodecFallback";
import {Terminal} from "xterm";
import {FitAddon} from 'xterm-addon-fit';

export class Connection {

    constructor(forceUpdate, canvasRef) {
        this.forceUpdate = forceUpdate;
        this.canvasRef = canvasRef;
        this.messages = [];
        this.kernMessages = [];
        this.started = false;
        this.shellOpen = false;
    }

    async start() {
        this.log("Starting");

        this.displayWidth = 720;
        this.displayHeight = 1280;

        this.canvas = this.canvasRef.current
        // this.canvas.width = this.displayWidth;
        // this.canvas.height = this.displayHeight;
        this.canvas.width = 320;
        this.canvas.height = 640;

        this.canvasContext = this.canvas.getContext('2d')
        this.canvasContext.fillStyle = "purple";
        this.canvasContext.fillRect(0, 0, this.canvas.width, this.canvas.height);

        // Mobile input
        this.canvas.addEventListener('touchstart', this.onStartDrag.bind(this));
        this.canvas.addEventListener('touchmove', this.onContinueDrag.bind(this));
        this.canvas.addEventListener('touchend', this.onEndDrag.bind(this));

        // Desktop input
        this.canvas.addEventListener('mousedown', this.onStartDrag.bind(this));
        this.canvas.addEventListener('mousemove', this.onContinueDrag.bind(this));
        this.canvas.addEventListener('mouseup', this.onEndDrag.bind(this));
        this.canvas.addEventListener('mouseout', this.onEndDrag.bind(this));

        if (isSecureContext) {
            this.log("Secure context");
        } else {
            this.log("WARNING: Non secure context");
        }

        if (await CheckWebCodecSupport()) {
            this.log("WebCodec Supported");
        } else {
            this.log("WARNING: WebCodec not native");
        }

        const config = {
            codec: "vp8",
            codedWidth: this.displayWidth,
            codedHeight: this.displayHeight,
            optimizeForLatency: true,
            // hardwareAcceleration: "prefer-hardware",
        };

        const {supported} = await window.VideoDecoder.isConfigSupported(config);
        if (supported) {
            this.log("Video supported");
        } else {
            this.log("ERROR: Video unsupported");
            return;
        }

        let loc = window.location, wsUrl;
        if (loc.protocol === "https:") {
            wsUrl = "wss:";
        } else {
            wsUrl = "ws:";
        }
        if (!process.env.NODE_ENV || process.env.NODE_ENV === 'development') {
            this.log("Detected development build - switching ports");
            wsUrl += "//" + loc.hostname + ":8080/ws123";
        } else {
            wsUrl += "//" + loc.host + "/ws";
        }

        this.log("Connecting to " + wsUrl);

        this.ws = new WebSocket(wsUrl);
        this.ws.onopen = this.wsOpen.bind(this);
        this.ws.onclose = this.wsClose.bind(this);
        this.ws.onerror = this.wsError.bind(this);
        this.ws.onmessage = this.wsMessage.bind(this);

        const init = {
            output: this.handleFrame.bind(this),
            error: this.handleDecodeError.bind(this),
        };
        this.decoder = new window.VideoDecoder(init);
        this.decoder.configure(config);
        // Unsupported in fallback
        // decoder.addEventListener("dequeue", (event) => {
        //     console.log(event);f
        // });
    }

    log(msg) {
        this.messages.push(msg);
        this.forceUpdate();
    }

    handleFrame(frame) {
        this.canvasContext.rect(0, 0, this.canvas.width, this.canvas.height);
        this.canvasContext.fillStyle = "red";
        this.canvasContext.fill();

        this.canvasContext.drawImage(frame, 0, 0, this.canvas.width, this.canvas.height);
        frame.close();
    }

    handleDecodeError(err) {
        this.log("decoder error: " + err.message);
    }

    wsOpen() {
        this.log("Connected to server");
    }

    wsClose() {
        this.log("Connection closed");
    }

    wsError() {
        this.log("Connected experienced an error");
    }

    wsMessage(messageEvent) {
        var wsMsg = messageEvent.data;
        if (typeof wsMsg === 'string') {
            let parsedMsg = JSON.parse(wsMsg);

            switch (parsedMsg.type) {
                case 'log':
                    this.log("< " + parsedMsg.line);
                    break;
                case 'kernlog':
                    this.kernMessages.push(parsedMsg.line);
                    this.forceUpdate();
                    break;
                case 'shell-data':
                    this.term.write(parsedMsg.line);
                    break;
            }
        } else {
            var arrayBuffer;
            var fileReader = new FileReader();
            fileReader.onload = async function (event) {
                arrayBuffer = event.target.result;
                var gdata = new Uint8Array(arrayBuffer);

                if (gdata[0] == 1) {
                    return;
                }

                var fdata = gdata.slice(2);
                var keyframe = gdata[1];

                const chunk = new window.EncodedVideoChunk({
                    timestamp: 0,
                    type: keyframe ? "key" : "delta",
                    data: fdata,
                });
                this.decoder.decode(chunk)
            }.bind(this);
            fileReader.readAsArrayBuffer(wsMsg);
        }
    }

    onStartDrag(event) {
        this.mouseDown = true;
        this.processTouchEvent(event);
    }

    onContinueDrag(event) {
        if (!this.mouseDown) {
            return;
        }

        this.processTouchEvent(event);
    }

    onEndDrag(event) {
        if (!this.mouseDown) {
            return;
        }

        this.mouseDown = false;
        this.processTouchEvent(event);
    }

    processTouchEvent(event) {
        let eventType = event.type.substring(0, 5);
        if (eventType == 'mouse') {
            this.sendTouchEvent(0, event.offsetX, event.offsetY, this.mouseDown ? 1 : 0);
        } else if (eventType == 'touch') {
            let changes = event.changedTouches;
            let rect = event.target.getBoundingClientRect();
            for (let i = 0; i < changes.length; i++) {

                this.sendTouchEvent(
                    changes[i].identifier,
                    changes[i].pageX - rect.left,
                    changes[i].pageY - rect.top,
                    this.mouseDown ? 1 : 0,
                );
            }
        }
    }

    sendTouchEvent(id, x, y, pressure) {
        const elementWidth = this.canvas.offsetWidth ? this.canvas.offsetWidth : 1;
        const elementHeight = this.canvas.offsetHeight ? this.canvas.offsetHeight : 1;
        const scalingX = this.displayWidth / elementWidth;
        const scalingY = this.displayHeight / elementHeight;
        x = Math.max(0, Math.min(this.displayWidth, Math.trunc(x * scalingX)));
        y = Math.max(0, Math.min(this.displayHeight, Math.trunc(y * scalingY)));

        const msg = {
            type: 'touch-event',
            touchEvent: {
                id: id,
                x: x,
                y: y,
                pressure: pressure,
            },
        };
        this.ws.send(JSON.stringify(msg));
    }

    openShell() {
        if (this.shellOpen) return;
        this.shellOpen = true;
        this.forceUpdate();

        this.term = new Terminal({
            scrollback: 1000,
        });
        const fitAddon = new FitAddon();
        this.term.loadAddon(fitAddon);
        this.term.open(document.getElementById('terminal'));
        fitAddon.fit();

        const msg = {
            type: 'open-shell',
        };
        this.ws.send(JSON.stringify(msg));

        this.term.onData(e => {
            const msg = {
                type: 'shell-data',
                line: e,
            };
            this.ws.send(JSON.stringify(msg));
        });
    }
}