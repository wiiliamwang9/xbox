#!/bin/bash

# TLS证书生成脚本 - mTLS双向认证
# 为Xbox Controller和Agent生成证书

set -e

CERT_DIR="./certs"
CA_DIR="$CERT_DIR/ca"
SERVER_DIR="$CERT_DIR/server"
CLIENT_DIR="$CERT_DIR/client"

# 证书配置
COUNTRY="CN"
STATE="Beijing"
CITY="Beijing"
ORG="Xbox"
OU="Xbox-TLS"
EMAIL="admin@xbox.local"

# 创建目录
mkdir -p "$CA_DIR" "$SERVER_DIR" "$CLIENT_DIR"

echo "=== 生成TLS证书用于mTLS双向认证 ==="

# 1. 生成CA私钥
echo "1. 生成CA私钥..."
openssl genrsa -out "$CA_DIR/ca-key.pem" 4096

# 2. 生成CA证书
echo "2. 生成CA证书..."
openssl req -new -x509 -days 3650 -key "$CA_DIR/ca-key.pem" -out "$CA_DIR/ca-cert.pem" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU-CA/CN=Xbox-CA/emailAddress=$EMAIL"

# 3. 生成服务器私钥
echo "3. 生成Controller服务器私钥..."
openssl genrsa -out "$SERVER_DIR/server-key.pem" 4096

# 4. 生成服务器证书请求
echo "4. 生成Controller服务器证书请求..."
openssl req -new -key "$SERVER_DIR/server-key.pem" -out "$SERVER_DIR/server.csr" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU-Server/CN=xbox-controller/emailAddress=$EMAIL"

# 5. 创建服务器证书扩展配置
echo "5. 创建服务器证书扩展配置..."
cat > "$SERVER_DIR/server-ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = xbox-controller
DNS.2 = localhost
DNS.3 = controller
IP.1 = 127.0.0.1
IP.2 = 0.0.0.0
IP.3 = ::1
EOF

# 6. 签署服务器证书
echo "6. 签署Controller服务器证书..."
openssl x509 -req -in "$SERVER_DIR/server.csr" -CA "$CA_DIR/ca-cert.pem" -CAkey "$CA_DIR/ca-key.pem" \
    -CAcreateserial -out "$SERVER_DIR/server-cert.pem" -days 365 -extfile "$SERVER_DIR/server-ext.cnf"

# 7. 生成客户端私钥
echo "7. 生成Agent客户端私钥..."
openssl genrsa -out "$CLIENT_DIR/client-key.pem" 4096

# 8. 生成客户端证书请求
echo "8. 生成Agent客户端证书请求..."
openssl req -new -key "$CLIENT_DIR/client-key.pem" -out "$CLIENT_DIR/client.csr" \
    -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$OU-Client/CN=xbox-agent/emailAddress=$EMAIL"

# 9. 创建客户端证书扩展配置
echo "9. 创建客户端证书扩展配置..."
cat > "$CLIENT_DIR/client-ext.cnf" << EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
EOF

# 10. 签署客户端证书
echo "10. 签署Agent客户端证书..."
openssl x509 -req -in "$CLIENT_DIR/client.csr" -CA "$CA_DIR/ca-cert.pem" -CAkey "$CA_DIR/ca-key.pem" \
    -CAcreateserial -out "$CLIENT_DIR/client-cert.pem" -days 365 -extfile "$CLIENT_DIR/client-ext.cnf"

# 11. 验证证书
echo "11. 验证证书..."
echo "验证服务器证书："
openssl verify -CAfile "$CA_DIR/ca-cert.pem" "$SERVER_DIR/server-cert.pem"
echo "验证客户端证书："
openssl verify -CAfile "$CA_DIR/ca-cert.pem" "$CLIENT_DIR/client-cert.pem"

# 12. 设置权限
echo "12. 设置证书文件权限..."
chmod 600 "$CA_DIR/ca-key.pem"
chmod 600 "$SERVER_DIR/server-key.pem"
chmod 600 "$CLIENT_DIR/client-key.pem"
chmod 644 "$CA_DIR/ca-cert.pem"
chmod 644 "$SERVER_DIR/server-cert.pem"
chmod 644 "$CLIENT_DIR/client-cert.pem"

# 13. 清理临时文件
echo "13. 清理临时文件..."
rm -f "$SERVER_DIR/server.csr" "$CLIENT_DIR/client.csr"
rm -f "$SERVER_DIR/server-ext.cnf" "$CLIENT_DIR/client-ext.cnf"

echo ""
echo "=== TLS证书生成完成！==="
echo "证书目录结构："
echo "certs/"
echo "├── ca/"
echo "│   ├── ca-cert.pem      # CA根证书"
echo "│   └── ca-key.pem       # CA私钥"
echo "├── server/"
echo "│   ├── server-cert.pem  # Controller服务器证书"
echo "│   └── server-key.pem   # Controller服务器私钥"
echo "└── client/"
echo "    ├── client-cert.pem  # Agent客户端证书"
echo "    └── client-key.pem   # Agent客户端私钥"
echo ""
echo "证书有效期："
echo "- CA证书: 10年"
echo "- 服务器/客户端证书: 1年"
echo ""
echo "配置说明："
echo "- Controller使用: ca-cert.pem, server-cert.pem, server-key.pem"
echo "- Agent使用: ca-cert.pem, client-cert.pem, client-key.pem"
echo "- mTLS双向认证已启用"