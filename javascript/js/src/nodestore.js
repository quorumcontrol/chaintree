const utils = require('./utils');

const nodestore = {};

nodestore.store = {};

nodestore.Get = (cid)=> {
    return this.store[cid];
}

nodestore.Store = (cid,node)=> {
    this.store[cid] = node;
}

module.exports = nodestore;