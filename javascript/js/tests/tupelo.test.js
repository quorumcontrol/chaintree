const chai = require('chai')
const expect = chai.expect

const utils = require('../src/utils');
const Tupelo = require('../src/tupelo');
const messages = require('../src/messages');

class FakeWorker{
    constructor(){}

    send(buf) {
        this.lastSend = buf;
    }

    recv(cb) {
        this.receiveCallback = cb;
    }

    testOnlyRemoteSend(buf) {
        this.receiveCallback(buf);
    }
}

describe('Tupelo', ()=> {
    it('can receive a start message', async ()=> {
        worker = new FakeWorker();
        tupelo = new Tupelo(worker);

        let obj = {foo: 'bar'};
        let serialized = await utils.serialize(obj);
        let cid = await utils.cidOfSerialized(serialized);
    
        let start = {
            tip: cid,
            nodes: [serialized],
        };

        let any = await messages.toAny("start", start);
        
        worker.testOnlyRemoteSend(any);
    });
});