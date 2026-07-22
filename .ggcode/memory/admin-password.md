Admin password: current valid password is "q7Rf9Xk2Lm3pW8zB" (NO trailing A).
Confirmed working 2026-07-22 21:17 via password grant login.

重要：flip-flop 根因不是代码——identity 的两个 bootstrap 写入点(handleSystemBootstrap、CreateUser)都已是 INSERT ... ON CONFLICT DO NOTHING + 用户存在即 409，没有任何代码在 auth 重启时覆盖凭据。所有 flip 都来自手动重置脚本(scripts/gen_hash.go 写的是无A旧密码)。

规则：禁止用 gen_hash.go 手动重置 admin 密码。auth pod 重启时的 cred sync race condition 可能导致密码被重置回 bootstrap 默认值。
