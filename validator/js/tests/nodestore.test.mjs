import chai from 'chai';
const expect = chai.expect;

import utils from '../src/utils.mjs';
import Nodestore from '../src/nodestore.mjs';

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
    });

    it('resolves a path that crosses nodes', async ()=> {
        let nodeStore = new Nodestore();
        let obj = {bar: 1};
        let serializedObj = await utils.serialize(obj);
        let objCID = await utils.cidOfSerialized(serializedObj);
        nodeStore.store(objCID, serializedObj);

        let root = {foo: objCID};
        let serializedRoot = await utils.serialize(root);
        let rootCID = await utils.cidOfSerialized(serializedRoot);
        nodeStore.store(rootCID, serializedRoot);

        let resp = await nodeStore.resolve(rootCID, "foo/bar");
        expect(resp.value).to.eql(1);
    });
});