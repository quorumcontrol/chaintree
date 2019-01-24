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
})