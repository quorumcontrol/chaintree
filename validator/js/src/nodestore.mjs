import utils from './utils.mjs';
import CID from 'cids';

class Nodestore {
    constructor() {
        this.storage = {};
    }

    get(cid) {
        return this.storage[cid.toBaseEncodedString()];
    }

    store(cid, node) {
        if (!CID.isCID(cid)) {
            console.log("converting cid: ", cid)
            cid = new CID(cid);
        }
        this.storage[cid.toBaseEncodedString()] = node;
    }

    async resolve(cid, path) {
        let blob = this.get(cid)
        let resp = await utils.resolve(blob, path);
        if (CID.isCID(resp.value) && resp.remainderPath.length > 0) {
            V8Worker2.print("resolve");
            return this.resolve(resp.value, resp.remainderPath);
        }
        return resp;
    }
}

export default Nodestore;