import dagCBOR from 'ipld-dag-cbor';
import Nodestore from  './nodestore.mjs';
import utils from './utils.mjs';
import messages from './messages.mjs';

function toArrayBuffer(buf) {
    var ab = new ArrayBuffer(buf.length);
    var view = new Uint8Array(ab);
    for (var i = 0; i < buf.length; ++i) {
        view[i] = buf[i];
    }
    return ab;
}

class Tupelo {
    constructor(webv8worker) {
        this.worker = webv8worker;
        this.nodestore = new Nodestore();
        this.worker.recv(this.receive.bind(this));
    }

    onStart(fn) {
        this.onStart = fn;
        if (this.started) {
            this.onStart(this);
        }
    }

    receive(buf) {
       let p = this.receiveMsg(buf);
       p.then(() => {}, (err) => {this.worker.print("errored")});
       return null;
    }

    async send(typeName, obj) {
        let any = await messages.toAny(typeName, obj);
        let bits = toArrayBuffer(any);
        this.worker.send(bits);
    }

    async receiveMsg(buf) {
        this.worker.print("receive")

        let newBuf = Buffer.from(buf);
        let any = await utils.deserialize(newBuf);
        let msg = await utils.deserialize(any.payload);
        
        switch(any.type){
        case "start":
            this.handleStart(msg);
            return;
        }
        
    }

    async handleStart(msg) {
        console.log(msg.nodes);
        this.worker.print("starting : ",msg.nodes);
        for (const k in msg.nodes) {
            this.worker.print("storing")
            this.nodestore.store(k, msg.nodes[k]);
            this.tip = msg.tip;
        }
        this.started = true;
        if (this.onStart) {
            this.worker.print("calling onStart")
            this.onStart(this);
        }
    }

}

export default Tupelo;