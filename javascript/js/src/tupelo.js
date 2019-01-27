import dagCBOR from 'ipld-dag-cbor';
import Nodestore from  './nodestore';
import utils from './utils';
import messages from './messages';

class Tupelo {
    constructor(webv8worker) {
        this.worker = webv8worker;
        this.nodestore = new Nodestore();
        this.worker.recv(this.receiveMsg.bind(this));
    }

    async receiveMsg(buf) {
        let any = await utils.deserialize(buf);
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
        this.worker.send(bits)
    }

}

export default Tupelo;