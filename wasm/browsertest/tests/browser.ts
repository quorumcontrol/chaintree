import { expect } from 'chai'

declare const Go: any;
declare const global: any;


describe("hello world", ()=> {
    it('works', ()=> {
        expect(true).to.be.true
    })

    it('loads a file', async ()=> {

        global.process = {
            pid: 2,
            title: window.navigator.userAgent
        };

        const wasmResp = await fetch("/main.wasm")
        const wasm = await wasmResp.arrayBuffer()
        const go = new Go();
        const result = await WebAssembly.instantiate(wasm, go.importObject)

        return go.run(result.instance)
    }).timeout(30000)
})