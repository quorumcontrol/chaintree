import chai from 'chai';
const expect = chai.expect;

import utils from '../src/utils.mjs';



describe('utils', ()=> {
    before(() => {
        global.V8Worker2 = {
            print: (arg1,arg2,arg3,arg4) => {
                // console.log.apply(null, [arg1,arg2,arg3,arg4]);
            }
        }
    });
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

    it('real-world deserializes', async ()=> {
        // let b64 = "omR0eXBlZXN0YXJ0Z3BheWxvYWRYP6JjdGlw2CpYJQABcRIgKwOmSdbOCT4qapCrRjj4ltQ2Kn+wk6xKLyqmiGEDg7xlbm9kZXOBSaFjZm9vY2Jhcg=="
        let b64 = "omR0eXBlZXN0YXJ0Z3BheWxvYWRYP6JjdGlw2CpYJQABcRIgKwOmSdbOCT4qapCrRjj4ltQ2Kn+wk6xKLyqmiGEDg7xlbm9kZXOBSaFjZm9vY2Jhcg=="
        let buf = Buffer.from(b64, "base64");
        let any = await utils.deserialize(buf);
        expect(any.type).to.eql("start");
        let msg = await utils.deserialize(any.payload);
        expect(msg.nodes.length).to.eql(1);
    })

    // it('calculates a cid from an object', async ()=> {
    //     let obj = {foo: 1};
    //     let cid = await utils.cid(obj);
    //     expect(cid.toBaseEncodedString()).to.eql("zdpuAo2cQJdBnUa3PorWFLWK7ijsFD2KRcs2YDRQ38pe1mQ8M");
    // });

    // it('calculates a cid from a serialized', async ()=> {
    //     let obj = {foo: 1};
    //     let serialized = await utils.serialize(obj);
    //     let cid = await utils.cidOfSerialized(serialized);
    //     expect(cid.toBaseEncodedString()).to.eql("zdpuAo2cQJdBnUa3PorWFLWK7ijsFD2KRcs2YDRQ38pe1mQ8M"); 
    // })
})