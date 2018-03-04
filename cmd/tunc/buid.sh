#!/usr/bin/env bash
/root/go/bin/gox --osarch="darwin/amd64"
/root/go/bin/gox --osarch="linux/386"
/root/go/bin/gox --osarch="linux/arm"
/root/go/bin/gox --osarch="linux/mips"
/root/go/bin/gox --osarch="windows/386"
/root/go/bin/gox --osarch="windows/amd64"
mv ./tuns.go ../
mv ./* ../../bin_data/static/clients