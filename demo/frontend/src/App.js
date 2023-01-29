import './App.css';
import React, { useRef, useEffect, useState, useReducer } from 'react'

import {Connection} from "./Connection";

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

    return (
        <div className="App">
            <header className="App-header">
                <canvas ref={canvasRef} className="ScreenCanvas" touch-action="none"></canvas>
                <h2>Log:</h2>
                <div className="LogOutput">
                    {messages}
                </div>
            </header>
        </div>
    );
}

export default App;
