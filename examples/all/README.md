## All In One

```
make

# https://kubernetes.io/zh/docs/reference/access-authn-authz/authentication/#static-token-file
echo 'token-1,system:yubo,uid:yubo,"group1,group2,group3"' > tokens.cvs

./helo \
	--config ./helo.yml \
	--token-auth-file=./tokens.cvs \
	-v 10 \
	--anonymous-auth \
	--authorization-mode=AlwaysAllow \
	--logtostderr
```
