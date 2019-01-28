const chai = require('chai');
const expect = chai.expect;
const dist = require("../dist/index.js");

describe('dist', ()=> {
    before(()=> {
        global.global = {};
        global.self = {};
    });

    it('works', ()=> {
        console.log(dist);
    });
})