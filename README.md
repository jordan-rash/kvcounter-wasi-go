Build  && Run steps

Requirements:
- [tinygo](https://github.com/tinygo-org/tinygo)
- [wit-deps](https://github.com/bytecodealliance/wit-deps)
- [wit-bindgen](https://github.com/bytecodealliance/wit-bindgen)
- [wasm-tools](https://github.com/bytecodealliance/wasm-tools)
- [just](https://github.com/casey/just) 
- [wasmcloud](https://github.com/wasmcloud/wasmcloud)
- [wash](https://github.com/wasmcloud/wash)

```
wit-deps
just build

export ACTOR_ID=<actorid>
export HOST_ID=<hostid> # of Rust host 

just start_actor
`
