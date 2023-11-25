import React, {useEffect, useReducer, useRef, useState} from 'react'
import Tab from 'react-bootstrap/Tab';
import Tabs from 'react-bootstrap/Tabs';
import {Connection} from "./Connection";
import 'xterm/css/xterm.css'
import './App.css';

function App() {
    const canvasRef = useRef(null);
    const [ignored, forceUpdate] = useReducer(x => x + 1, 0);

    let [state, setState] = useState(0);
    if (state === 0) {
        state = new Connection(forceUpdate, canvasRef);
        setState(state);
    }

    useEffect(() => {
        if (state.started) {
            return;
        }
        state.started = true;
        state.start();
    }, []);

    const messages = state.messages.map((msg, index) => (
        <div key={index}>{msg}</div>
    )).reverse();

    const kernMessages = state.kernMessages.map((msg, index) => (
        <div key={index}>{msg}</div>
    )).reverse();

    function handleShellOpen(e) {
        e.preventDefault();
        state.openShell();
    }

    function handleStart(e) {
        e.preventDefault();
        state.sendPowerEvent("start");
    }

    function handleStop(e) {
        e.preventDefault();
        state.sendPowerEvent("stop");
    }

    function handleForceStop(e) {
        e.preventDefault();
        state.sendPowerEvent("force-stop");
    }

    return (
        <div className="App">
            <header className="App-header">
                <Tabs
                    defaultActiveKey="display"
                    id="uncontrolled-tab-example"
                >
                    <Tab eventKey="display" title="Display">
                        <button onClick={handleStart}>
                            Start
                        </button>
                        <button onClick={handleStop}>
                            Stop
                        </button>
                        <button onClick={handleForceStop}>
                            Force Stop
                        </button>
                        <br/>
                        <canvas ref={canvasRef} className="ScreenCanvas" touch-action="none"></canvas>
                    </Tab>
                    <Tab eventKey="log" title="Log">
                        <div className="LogOutput">
                            {messages}
                        </div>
                    </Tab>
                    <Tab eventKey="kernlog" title="KernLog">
                        <div className="LogOutput">
                            {kernMessages}
                        </div>
                    </Tab>
                    <Tab eventKey="shell" title="Shell">
                        <button onClick={handleShellOpen} disabled={state.shellOpen}>
                            Open Shell
                        </button>
                        <div className="terminal-wrapper">
                            <div id="terminal"></div>
                        </div>
                    </Tab>
                </Tabs>


            </header>
        </div>
    );
}

export default App;
