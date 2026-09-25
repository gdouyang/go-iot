package license

// DefaultPublicKeyPEM 内置默认验签公钥 (Ed25519)
// 生产环境可由 conf/app.yaml 中 license.public-key 配置覆盖
const DefaultPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MCowBQYDK2VwAyEAqFrvnR5FJlcaul7uBYD93iCbqgn/G6yne1PLR71BW9E=
-----END PUBLIC KEY-----`
