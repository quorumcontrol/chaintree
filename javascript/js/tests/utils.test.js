const chai = require('chai')
const expect = chai.expect

const utils = require('../src/utils');

describe('utils', ()=> {
    it('serializes', async ()=> {
        let obj = {foo: 1};
        let serialized = await utils.serialize(obj);
        expect(serialized.length).to.equal(6);
    });

    it('deserializes', async ()=> {
        let obj = {foo: 1};
        let serialized = await utils.serialize(obj);

        let deserialized = await utils.deserialize(serialized);
        expect(deserialized).to.eql(obj);
    });

    it('calculates a cid from an object', async ()=> {
        let obj = {foo: 1};
        let cid = await utils.cid(obj);
        expect(cid.toBaseEncodedString()).to.eql("zdpuAo2cQJdBnUa3PorWFLWK7ijsFD2KRcs2YDRQ38pe1mQ8M");
    });

    it('calculates a cid from a serialized', async ()=> {
        let obj = {foo: 1};
        let serialized = await utils.serialize(obj);
        let cid = await utils.cidOfSerialized(serialized);
        expect(cid.toBaseEncodedString()).to.eql("zdpuAo2cQJdBnUa3PorWFLWK7ijsFD2KRcs2YDRQ38pe1mQ8M"); 
    })
})