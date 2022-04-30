## All In One

```
make

# https://kubernetes.io/zh/docs/reference/access-authn-authz/authentication/#static-token-file
echo 'token-1,system:yubo,uid:yubo,"group1,group2,group3"' > tokens.cvs

./all-in-one \
	-f ./etc/config.yaml \
	--token-auth-file=./tokens.cvs \
	-v 10 \
	--anonymous-auth \
	--authorization-mode=AlwaysAllow \
	--logtostderr
```


#### debug
```
# print config loaded
./all-in-on  -f ./etc/config.yaml --debug-config --dry-run

# debug flags set
./all-in-on  -f ./etc/config.yaml --debug-flags --dry-run
```
