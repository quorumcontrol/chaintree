import chai from 'chai';
const expect = chai.expect

import utils from '../src/utils.mjs';
import Tupelo from '../src/tupelo.mjs';
import messages from '../src/messages.mjs';

class FakeWorker{
    constructor(){}

    send(buf) {
        this.lastSend = buf;
    }

    recv(cb) {
        this.receiveCallback = cb;
    }

    print(args) {
        console.log.apply(null, arguments);
    }

    testOnlyRemoteSend(buf) {
        this.receiveCallback(buf);
    }
}

describe('Tupelo', ()=> {
    function toArrayBuffer(buf) {
        var ab = new ArrayBuffer(buf.length);
        var view = new Uint8Array(ab);
        for (var i = 0; i < buf.length; ++i) {
            view[i] = buf[i];
        }
        return ab;
    }

    before(() => {
        global.V8Worker2 = {
            print: (arg1, arg2  ) => {
                // console.log(arg1, arg2);
            }
        }
    });
    it('can receive a start message', async ()=> {
        let worker = new FakeWorker();
        let tupelo = new Tupelo(worker);
        let b64 = "omR0eXBlZXN0YXJ0Z3BheWxvYWRYP6JjdGlw2CpYJQABcRIgKwOmSdbOCT4qapCrRjj4ltQ2Kn+wk6xKLyqmiGEDg7xlbm9kZXOBSaFjZm9vY2Jhcg=="
        let buf = Buffer.from(b64, "base64");
        worker.testOnlyRemoteSend(toArrayBuffer(buf));
    });
});