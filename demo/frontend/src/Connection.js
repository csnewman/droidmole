import {CheckWebCodecSupport} from "./CodecFallback";

export class Connection {

    constructor(forceUpdate, canvasRef) {
        this.forceUpdate = forceUpdate;
        this.canvasRef = canvasRef;
        this.messages = [];
        this.started = false;
        this.touchIdSlotMap = new Map();
        this.touchSlots = [];
    }

    async start() {
        this.log("Starting");

        this.canvas = this.canvasRef.current
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
            // codedWidth: 720,
            // codedHeight: 1280,
            codedWidth: 1920,
            codedHeight: 1080,
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
        // console.log(decoder.addEventListener);
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
            this.log("Received: " + wsMsg);
        } else {
            var arrayBuffer;
            var fileReader = new FileReader();
            fileReader.onload = async function(event) {
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
            };
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
        const deviceDisplay = event.target;

        let xArr = [];
        let yArr = [];
        let slotArr = [];

        if (eventType == 'mouse') {
            xArr.push(event.offsetX);
            yArr.push(event.offsetY);
            slotArr.push(0);
        } else if (eventType == 'touch') {
            let changes = event.changedTouches;
            let rect = event.target.getBoundingClientRect();
            for (let i = 0; i < changes.length; i++) {
                xArr.push(changes[i].pageX - rect.left);
                yArr.push(changes[i].pageY - rect.top);
                if (this.touchIdSlotMap.has(changes[i].identifier)) {
                    let slot = this.touchIdSlotMap.get(changes[i].identifier);

                    slotArr.push(slot);
                    if (event.type == 'touchend') {
                        this.touchSlots[slot] = false;
                        this.touchIdSlotMap.delete(changes[i].identifier);
                    }
                } else if (event.type == 'touchstart') {
                    let slot = -1;
                    for (let i = 0; i < this.touchSlots.length; i++) {
                        if (!this.touchSlots[i]) {
                            slot = i;
                            break;
                        }
                    }

                    if (slot == -1) {
                        slot = this.touchSlots.length;
                        this.touchSlots.push(true);
                    }

                    slotArr.push(slot);
                    this.touchSlots[slot] = true;
                    this.touchIdSlotMap.set(changes[i].identifier, slot);
                }
            }
        }

        const screenWidth = 1080;
        const screenHeight = 1920;
        const elementWidth = deviceDisplay.offsetWidth ? deviceDisplay.offsetWidth : 1;
        const elementHeight = deviceDisplay.offsetHeight ? deviceDisplay.offsetHeight : 1;

        const scalingX = screenWidth / elementWidth;
        const scalingY = screenHeight / elementHeight;

        for (let i = 0; i < xArr.length; i++) {
            xArr[i] = Math.max(0, Math.min(screenWidth, Math.trunc(xArr[i] * scalingX)));
            yArr[i] = Math.max(0, Math.min(screenHeight, Math.trunc(yArr[i] * scalingY)));
        }

        const msg = {
            type: 'touch',
            x: xArr,
            y: yArr,
            down: this.mouseDown ? 1 : 0,
            slot: slotArr,
        };
        this.ws.send(JSON.stringify(msg));
    }
}