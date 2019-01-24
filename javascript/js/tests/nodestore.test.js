const chai = require('chai')
const expect = chai.expect

const utils = require('../src/utils');
const Nodestore = require('../src/nodestore');

describe('nodestore', ()=> {
    it('stores', async ()=>{
        let nodeStore = new Nodestore();
        let obj = {bar: 1};
        let serialized = await utils.serialize(obj);
        let cid = await utils.cidOfSerialized(serialized);
        nodeStore.store(cid, serialized);
        expect(nodeStore.get(cid)).to.eql(serialized);
    });

    it('resolves a single path', async ()=> {
        let nodeStore = new Nodestore();
        let obj = {bar: 1};
        let serialized = await utils.serialize(obj);
        let cid = await utils.cidOfSerialized(serialized);
        nodeStore.store(cid, serialized);
        let resp = await nodeStore.resolve(cid, "bar");
        expect(resp.value).to.eql(1);
    })
});