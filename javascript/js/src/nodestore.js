const utils = require('./utils');
const CID = require('cids')

class Nodestore {
    constructor() {
        this.storage = {};
    }

    get(cid) {
        return this.storage[cid.toBaseEncodedString()];
    }

    store(cid, node) {
        this.storage[cid.toBaseEncodedString()] = node;
    }

    async resolve(cid, path) {
        let blob = this.get(cid)
        let resp = await utils.resolve(blob, path)
        if (CID.isCID(resp.value) && resp.remainderPath.length > 0) {
            return this.resolve(resp.value, resp.remainderPath);
        }
        return resp;
    }
}

module.exports = Nodestore;