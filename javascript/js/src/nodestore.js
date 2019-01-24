const utils = require('./utils');

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
}

module.exports = Nodestore;