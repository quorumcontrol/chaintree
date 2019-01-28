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

    receive(buf) {
       let p = this.receiveMsg(buf);
       this.worker.print("receiveMsg return: ", p);
       p.then(() => {}, (err) => {this.worker.print("errored")});
       return null;
    }

    send(buf) {
        let bits = toArrayBuffer(buf);
        this.worker.send(bits);
    }

    async receiveMsg(buf) {
        let newBuf = Buffer.from(buf);
        this.worker.print("receive ", newBuf.toString('base64'));

        let any = await utils.deserialize(newBuf);
        let msg = await utils.deserialize(any.payload);
        
        switch(any.type){
        case "start":
            this.handleStart(msg);
            return;
        }
        
    }

    async handleStart(msg) {
        let bits = await messages.toAny({
            result: "ok",
        });
        this.send(bits);
    }

}

export default Tupelo;