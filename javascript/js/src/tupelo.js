const dagCBOR = require('ipld-dag-cbor');
const Nodestore = require('./nodestore');
const utils = require('./utils');
const messages = require('./messages');

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



module.exports = Tupelo;