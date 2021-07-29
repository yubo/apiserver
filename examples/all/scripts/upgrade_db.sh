#!/usr/bin/env bash
OLDPATH=`pwd`
ROOT=`cd $(dirname $0)../; pwd`
cd ${ROOT}
function finish {
    cd ${OLDPATH} 
}
trap finish EXIT


DB=sso
TMP_FILE=/tmp/${DB}_diff.sql
TMP_DB=${DB}_tmp

echo "drop database if exists ${TMP_DB}" | mysql
echo "create database ${TMP_DB}" | mysql
mysql ${TMP_DB} < misc/sample.mysql.sql
mysqldiff --dsn1="${MYSQL_USER}:${MYSQL_PWD}@tcp(localhost:3306)/${DB}?charset=utf8" \
	--dsn2="${MYSQL_USER}:${MYSQL_PWD}@tcp(localhost:3306)/${TMP_DB}?charset=utf8" \
	--logtostderr > ${TMP_FILE}


echo "drop database ${TMP_DB}" | mysql
cat ${TMP_FILE}
echo "--------"
echo "mysql ${DB} < ${TMP_FILE}"
echo

