# Changelog

All notable changes to this project are documented here, newest first.

Entries are generated from [Conventional Commits](https://www.conventionalcommits.org)
and grouped by change type. This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### CI & Build

- Bump care to v0.8.1 by [@iberflow](https://github.com/iberflow) in [65858b8](https://github.com/toaweme/http/commit/65858b88f1e95b021f751f71fc940deaa774f965).
- Use stable go for release gate to avoid old-go.mod tool-install failures by [@iberflow](https://github.com/iberflow) in [39d313e](https://github.com/toaweme/http/commit/39d313e4a4cda97d312bddc6a838fdec93ba0b0a).

## [0.2.1] - 2026-07-01

### CI & Build

- Bump care to v0.8.0 by [@iberflow](https://github.com/iberflow) in [7812ab1](https://github.com/toaweme/http/commit/7812ab17ab037da52908599cc31ddd5923dc21c8).
- Bump care to v0.7.1 and pin to commit sha by [@iberflow](https://github.com/iberflow) in [e76b7da](https://github.com/toaweme/http/commit/e76b7da6004575cb01c6408bf957afa384174f19).
- Bump care to v0.6.0 and fix card-svg dark/light wiring by [@iberflow](https://github.com/iberflow) in [10a3455](https://github.com/toaweme/http/commit/10a3455f8795021a16c40d7b9a238c1b0608ef18).

### Chores & Other

- Add CHANGELOG and bump care quality gate to v0.4.0 by [@iberflow](https://github.com/iberflow) in [3d0577f](https://github.com/toaweme/http/commit/3d0577f76da8385f262bdb41246f49f7eef6b416).

## [0.2.0] - 2026-06-27

### Features

- Unbuffered streaming responses via Request.Stream by [@iberflow](https://github.com/iberflow) in [5712570](https://github.com/toaweme/http/commit/57125709f20b23b0662665341eb3b4bc33dd4092).

### CI & Build

- Simplify quality workflow by [@iberflow](https://github.com/iberflow) in [075deec](https://github.com/toaweme/http/commit/075deec6208a8c9d56e1da044056dd4f0b652fe6).
- Lean quality.yml, let care run the checks by [@iberflow](https://github.com/iberflow) in [382c94e](https://github.com/toaweme/http/commit/382c94e4e42e220d1d0a0199665ed453638e1503).

### Chores & Other

- Fix server readme by [@iberflow](https://github.com/iberflow) in [9ae6879](https://github.com/toaweme/http/commit/9ae6879cf01e47e90e1ee8510c225d4104f02e07).
- Align README and CI with org standards by [@iberflow](https://github.com/iberflow) in [26ae400](https://github.com/toaweme/http/commit/26ae400b87aae444a25f44118a56aa4c109a9c03).

## [0.1.1] - 2026-06-15

### Fixes

- Ci by [@iberflow](https://github.com/iberflow) in [6d1e085](https://github.com/toaweme/http/commit/6d1e085c909de4134974c289c230b735316408ee).

### Chores & Other

- Update readme by [@iberflow](https://github.com/iberflow) in [6a2eb9e](https://github.com/toaweme/http/commit/6a2eb9e9b9e046079f99e5d1c3172eef60fe6bca).

## [0.1.0] - 2026-06-15

### Features

- Tracing headers, url without baseURL by [@iberflow](https://github.com/iberflow) in [6015b0e](https://github.com/toaweme/http/commit/6015b0ec575051fafdb83f8831631927e566455c).
- Configurable logging by [@iberflow](https://github.com/iberflow) in [75821b7](https://github.com/toaweme/http/commit/75821b776c29ce35780e2a77710d23246447917f).
- Add ctx to methods by [@iberflow](https://github.com/iberflow) in [ca020c3](https://github.com/toaweme/http/commit/ca020c3d6a533b7940d29127bab2e10e03c627e5).
- Sse support by [@iberflow](https://github.com/iberflow) in [1ba2b42](https://github.com/toaweme/http/commit/1ba2b42b10da978253525c176a16382ae6a53f89).
- Stdlib chi by [@iberflow](https://github.com/iberflow) in [c8a064e](https://github.com/toaweme/http/commit/c8a064ee56891e583f68b8c69ea673d428e87452).
- **Server:** Split root and inline middleware chains, add CORS by [@iberflow](https://github.com/iberflow) in [5ef1760](https://github.com/toaweme/http/commit/5ef17605bdc98bd8b85f8c4526d89a39c2b720e7).
- Url param functions by [@iberflow](https://github.com/iberflow) in [9cfadd2](https://github.com/toaweme/http/commit/9cfadd2a5945bb0e53bd8cfa2a27cc4f38f7fb9f).
- Log routes by [@iberflow](https://github.com/iberflow) in [6147e09](https://github.com/toaweme/http/commit/6147e09f55f941563a3ff1681a2286d7e9965835).
- Log routes by [@iberflow](https://github.com/iberflow) in [5aaeb57](https://github.com/toaweme/http/commit/5aaeb57fe8b58298032eaf42b818f513db7fc449).
- Logging middleware by [@iberflow](https://github.com/iberflow) in [5c531d8](https://github.com/toaweme/http/commit/5c531d8467d285fc7009d65624f18f49c0b3dc6e).
- Sse by [@iberflow](https://github.com/iberflow) in [752efce](https://github.com/toaweme/http/commit/752efcee2210d395d6f26892ad48320a14971130).
- Ditch logger dep + client tests and benchmarks by [@iberflow](https://github.com/iberflow) in [ba74165](https://github.com/toaweme/http/commit/ba741659e487f1f3fe244ceebb08dbb8e5277c3c).
- Server options and tests by [@iberflow](https://github.com/iberflow) in [8849c93](https://github.com/toaweme/http/commit/8849c9390beab921e9400af2a5803d61a671984a).
- Client and server readme by [@iberflow](https://github.com/iberflow) in [5119a2e](https://github.com/toaweme/http/commit/5119a2ead1b8e544a5de953d8e18d026f42fe75b).
- **Ci:** Run benchmarks by [@iberflow](https://github.com/iberflow) in [844618b](https://github.com/toaweme/http/commit/844618b78d57a6a11045c981b7726ed2302ed499).

### Fixes

- Server start error handling by [@iberflow](https://github.com/iberflow) in [c0301e2](https://github.com/toaweme/http/commit/c0301e2f41627698e429d47f4d9b2a069697e85f).
- Limit body print by [@iberflow](https://github.com/iberflow) in [f292c50](https://github.com/toaweme/http/commit/f292c500d8bb8789811ea68067936c6eb4da5c80).
- Logging by [@iberflow](https://github.com/iberflow) in [81d5988](https://github.com/toaweme/http/commit/81d59889e4ac4ffdda2eec19bd2f55bbf979c27e).
- Logging by [@iberflow](https://github.com/iberflow) in [b87e716](https://github.com/toaweme/http/commit/b87e716e2d377111947cf8ca5043287e7aeffb24).
- Sse doStream message parsing by [@iberflow](https://github.com/iberflow) in [9b70eeb](https://github.com/toaweme/http/commit/9b70eeb3076c7f4742f503d5531c427fd0ada2b1).
- Sse client by [@iberflow](https://github.com/iberflow) in [8d835d4](https://github.com/toaweme/http/commit/8d835d48b174c6d51e2371ef2de47ec1ba08f7ed).
- Sse by [@iberflow](https://github.com/iberflow) in [6f2fab1](https://github.com/toaweme/http/commit/6f2fab1b35df22eabca2b5d3202747515b515709).
- Logging by [@iberflow](https://github.com/iberflow) in [ff65546](https://github.com/toaweme/http/commit/ff655464652ef07d7f556c7237d4e17f8bd0b32d).
- **Ci:** Build matrix by [@iberflow](https://github.com/iberflow) in [d1c466f](https://github.com/toaweme/http/commit/d1c466f07d12f29f8466a95e83c46d2b2473c4fa).

### Refactors

- Modules by [@iberflow](https://github.com/iberflow) in [e2e0453](https://github.com/toaweme/http/commit/e2e04531386f21cb56ac20a103d49e14907fe14c).

### Chores & Other

- Initial commit :) by [@iberflow](https://github.com/iberflow) in [dce70b7](https://github.com/toaweme/http/commit/dce70b714afc595cadd18944fb808cfc90e776f8).
- Leave only client by [@iberflow](https://github.com/iberflow) in [4e5a84a](https://github.com/toaweme/http/commit/4e5a84a4ff2866de0a57b7789f2dfbbc6333e5f6).
- Bump log module by [@iberflow](https://github.com/iberflow) in [9a126c4](https://github.com/toaweme/http/commit/9a126c4ed91235c68cc8e35a11c9230307d655ad).
- Bump log module by [@iberflow](https://github.com/iberflow) in [de191ac](https://github.com/toaweme/http/commit/de191ac893f5f6cb85ac6637188f937b525e3233).
- Tidy up by [@iberflow](https://github.com/iberflow) in [ba19aeb](https://github.com/toaweme/http/commit/ba19aeb9ed48e270b6b8ddf44750015fa00c31b0).
- Bump deps by [@iberflow](https://github.com/iberflow) in [f7cdb38](https://github.com/toaweme/http/commit/f7cdb38a262aba3d3e8cbe4a2a5b3f2b9a2fedb6).
- Enable full http client data logging by [@iberflow](https://github.com/iberflow) in [dba31a8](https://github.com/toaweme/http/commit/dba31a8efdc202326ed4000ba87e9cceb0613965).
- Bump logger by [@iberflow](https://github.com/iberflow) in [dd640d9](https://github.com/toaweme/http/commit/dd640d9d4b903d0e0434e7a7157016ec88bdacc8).
- Cleanup by [@iberflow](https://github.com/iberflow) in [aea82ee](https://github.com/toaweme/http/commit/aea82eee793ebba2e51b2fb006f90593261683a7).
- Cleanup by [@iberflow](https://github.com/iberflow) in [13f2bd4](https://github.com/toaweme/http/commit/13f2bd4f34b15c79ec25e6ea8275908dc91eef9a).

## [server/v0.2.0] - 2026-06-27

### Features

- Unbuffered streaming responses via Request.Stream by [@iberflow](https://github.com/iberflow) in [5712570](https://github.com/toaweme/http/commit/57125709f20b23b0662665341eb3b4bc33dd4092).

### CI & Build

- Simplify quality workflow by [@iberflow](https://github.com/iberflow) in [075deec](https://github.com/toaweme/http/commit/075deec6208a8c9d56e1da044056dd4f0b652fe6).
- Lean quality.yml, let care run the checks by [@iberflow](https://github.com/iberflow) in [382c94e](https://github.com/toaweme/http/commit/382c94e4e42e220d1d0a0199665ed453638e1503).

### Chores & Other

- Fix server readme by [@iberflow](https://github.com/iberflow) in [9ae6879](https://github.com/toaweme/http/commit/9ae6879cf01e47e90e1ee8510c225d4104f02e07).
- Align README and CI with org standards by [@iberflow](https://github.com/iberflow) in [26ae400](https://github.com/toaweme/http/commit/26ae400b87aae444a25f44118a56aa4c109a9c03).

## [server/v0.1.1] - 2026-06-15

### Fixes

- Ci by [@iberflow](https://github.com/iberflow) in [6d1e085](https://github.com/toaweme/http/commit/6d1e085c909de4134974c289c230b735316408ee).

### Chores & Other

- Update readme by [@iberflow](https://github.com/iberflow) in [6a2eb9e](https://github.com/toaweme/http/commit/6a2eb9e9b9e046079f99e5d1c3172eef60fe6bca).

## [server/v0.1.0] - 2026-06-15

### Features

- Tracing headers, url without baseURL by [@iberflow](https://github.com/iberflow) in [6015b0e](https://github.com/toaweme/http/commit/6015b0ec575051fafdb83f8831631927e566455c).
- Configurable logging by [@iberflow](https://github.com/iberflow) in [75821b7](https://github.com/toaweme/http/commit/75821b776c29ce35780e2a77710d23246447917f).
- Add ctx to methods by [@iberflow](https://github.com/iberflow) in [ca020c3](https://github.com/toaweme/http/commit/ca020c3d6a533b7940d29127bab2e10e03c627e5).
- Sse support by [@iberflow](https://github.com/iberflow) in [1ba2b42](https://github.com/toaweme/http/commit/1ba2b42b10da978253525c176a16382ae6a53f89).
- Stdlib chi by [@iberflow](https://github.com/iberflow) in [c8a064e](https://github.com/toaweme/http/commit/c8a064ee56891e583f68b8c69ea673d428e87452).
- **Server:** Split root and inline middleware chains, add CORS by [@iberflow](https://github.com/iberflow) in [5ef1760](https://github.com/toaweme/http/commit/5ef17605bdc98bd8b85f8c4526d89a39c2b720e7).
- Url param functions by [@iberflow](https://github.com/iberflow) in [9cfadd2](https://github.com/toaweme/http/commit/9cfadd2a5945bb0e53bd8cfa2a27cc4f38f7fb9f).
- Log routes by [@iberflow](https://github.com/iberflow) in [6147e09](https://github.com/toaweme/http/commit/6147e09f55f941563a3ff1681a2286d7e9965835).
- Log routes by [@iberflow](https://github.com/iberflow) in [5aaeb57](https://github.com/toaweme/http/commit/5aaeb57fe8b58298032eaf42b818f513db7fc449).
- Logging middleware by [@iberflow](https://github.com/iberflow) in [5c531d8](https://github.com/toaweme/http/commit/5c531d8467d285fc7009d65624f18f49c0b3dc6e).
- Sse by [@iberflow](https://github.com/iberflow) in [752efce](https://github.com/toaweme/http/commit/752efcee2210d395d6f26892ad48320a14971130).
- Ditch logger dep + client tests and benchmarks by [@iberflow](https://github.com/iberflow) in [ba74165](https://github.com/toaweme/http/commit/ba741659e487f1f3fe244ceebb08dbb8e5277c3c).
- Server options and tests by [@iberflow](https://github.com/iberflow) in [8849c93](https://github.com/toaweme/http/commit/8849c9390beab921e9400af2a5803d61a671984a).
- Client and server readme by [@iberflow](https://github.com/iberflow) in [5119a2e](https://github.com/toaweme/http/commit/5119a2ead1b8e544a5de953d8e18d026f42fe75b).
- **Ci:** Run benchmarks by [@iberflow](https://github.com/iberflow) in [844618b](https://github.com/toaweme/http/commit/844618b78d57a6a11045c981b7726ed2302ed499).

### Fixes

- Server start error handling by [@iberflow](https://github.com/iberflow) in [c0301e2](https://github.com/toaweme/http/commit/c0301e2f41627698e429d47f4d9b2a069697e85f).
- Limit body print by [@iberflow](https://github.com/iberflow) in [f292c50](https://github.com/toaweme/http/commit/f292c500d8bb8789811ea68067936c6eb4da5c80).
- Logging by [@iberflow](https://github.com/iberflow) in [81d5988](https://github.com/toaweme/http/commit/81d59889e4ac4ffdda2eec19bd2f55bbf979c27e).
- Logging by [@iberflow](https://github.com/iberflow) in [b87e716](https://github.com/toaweme/http/commit/b87e716e2d377111947cf8ca5043287e7aeffb24).
- Sse doStream message parsing by [@iberflow](https://github.com/iberflow) in [9b70eeb](https://github.com/toaweme/http/commit/9b70eeb3076c7f4742f503d5531c427fd0ada2b1).
- Sse client by [@iberflow](https://github.com/iberflow) in [8d835d4](https://github.com/toaweme/http/commit/8d835d48b174c6d51e2371ef2de47ec1ba08f7ed).
- Sse by [@iberflow](https://github.com/iberflow) in [6f2fab1](https://github.com/toaweme/http/commit/6f2fab1b35df22eabca2b5d3202747515b515709).
- Logging by [@iberflow](https://github.com/iberflow) in [ff65546](https://github.com/toaweme/http/commit/ff655464652ef07d7f556c7237d4e17f8bd0b32d).
- **Ci:** Build matrix by [@iberflow](https://github.com/iberflow) in [d1c466f](https://github.com/toaweme/http/commit/d1c466f07d12f29f8466a95e83c46d2b2473c4fa).

### Refactors

- Modules by [@iberflow](https://github.com/iberflow) in [e2e0453](https://github.com/toaweme/http/commit/e2e04531386f21cb56ac20a103d49e14907fe14c).

### Chores & Other

- Initial commit :) by [@iberflow](https://github.com/iberflow) in [dce70b7](https://github.com/toaweme/http/commit/dce70b714afc595cadd18944fb808cfc90e776f8).
- Leave only client by [@iberflow](https://github.com/iberflow) in [4e5a84a](https://github.com/toaweme/http/commit/4e5a84a4ff2866de0a57b7789f2dfbbc6333e5f6).
- Bump log module by [@iberflow](https://github.com/iberflow) in [9a126c4](https://github.com/toaweme/http/commit/9a126c4ed91235c68cc8e35a11c9230307d655ad).
- Bump log module by [@iberflow](https://github.com/iberflow) in [de191ac](https://github.com/toaweme/http/commit/de191ac893f5f6cb85ac6637188f937b525e3233).
- Tidy up by [@iberflow](https://github.com/iberflow) in [ba19aeb](https://github.com/toaweme/http/commit/ba19aeb9ed48e270b6b8ddf44750015fa00c31b0).
- Bump deps by [@iberflow](https://github.com/iberflow) in [f7cdb38](https://github.com/toaweme/http/commit/f7cdb38a262aba3d3e8cbe4a2a5b3f2b9a2fedb6).
- Enable full http client data logging by [@iberflow](https://github.com/iberflow) in [dba31a8](https://github.com/toaweme/http/commit/dba31a8efdc202326ed4000ba87e9cceb0613965).
- Bump logger by [@iberflow](https://github.com/iberflow) in [dd640d9](https://github.com/toaweme/http/commit/dd640d9d4b903d0e0434e7a7157016ec88bdacc8).
- Cleanup by [@iberflow](https://github.com/iberflow) in [aea82ee](https://github.com/toaweme/http/commit/aea82eee793ebba2e51b2fb006f90593261683a7).
- Cleanup by [@iberflow](https://github.com/iberflow) in [13f2bd4](https://github.com/toaweme/http/commit/13f2bd4f34b15c79ec25e6ea8275908dc91eef9a).

[Unreleased]: https://github.com/toaweme/http/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/toaweme/http/compare/server/v0.2.0...v0.2.1
[0.2.0]: https://github.com/toaweme/http/compare/server/v0.1.1...v0.2.0
[0.1.1]: https://github.com/toaweme/http/compare/server/v0.1.0...v0.1.1
[0.1.0]: https://github.com/toaweme/http/releases/tag/v0.1.0
[server/v0.2.0]: https://github.com/toaweme/http/compare/server/v0.1.1...server/v0.2.0
[server/v0.1.1]: https://github.com/toaweme/http/compare/server/v0.1.0...server/v0.1.1
[server/v0.1.0]: https://github.com/toaweme/http/releases/tag/server/v0.1.0
