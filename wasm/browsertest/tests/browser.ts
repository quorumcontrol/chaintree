import { expect } from 'chai'

declare const Go: any;


describe("hello world", ()=> {
    it('works', ()=> {
        expect(true).to.be.true
    })

    it('loads a file', async ()=> {
        const wasmResp = await fetch("/main.wasm")
        const wasm = await wasmResp.arrayBuffer()
        const go = new Go();
        const result = await WebAssembly.instantiate(wasm, go.importObject)

        return go.run(result.instance)
    })
})