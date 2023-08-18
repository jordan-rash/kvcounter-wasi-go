Build steps
```
└─❯ tinygo build --target=wasi .    
└─❯ wasm-tools component embed --world kvcounter ./wit kvcounter-wasi.wasm -o kvcounter-wasi.embed.wasm
└─❯ wasm-tools component new kvcounter-wasi.embed.wasm --adapt wasi_snapshot_preview1=./wasi_snapshot_preview1.wasm -o kvcounter-wasi.component.wasm 
```
