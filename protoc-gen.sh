#!/bin/bash

echo -e '🔨 Generating Golang code ...\n'
protoc -I ./backend ./backend/api/proto/v1/*.proto --go_out=plugins=grpc:./backend 
echo -e '🔨 Generating Swagger specs ...\n'
protoc -I ./backend  ./backend/api/proto/v1/*.proto --swagger_out=logtostderr=true:./backend
echo -e '🔨 Generating GRPC-Gateway ...\n'
protoc -I ./backend ./backend/api/proto/v1/*.proto --grpc-gateway_out=logtostderr=true:./backend

# Javascript code gen via grpc_web
cd ./backend

export GOOGLE_PROTO_DIR_IN=google
export SWAGGER_PROTO_DIR_IN=protoc-gen-swagger
export API_DIR_IN=api
export CLI_SRC=../../unpaper_cli/src

echo '🔨 Generating Javascript code ...'
dirs=("${GOOGLE_PROTO_DIR_IN}" "${SWAGGER_PROTO_DIR_IN}" "${API_DIR_IN}")
for d in "${dirs[@]}"; do
  for f in $(find "${d}" -name '*.proto'); do
    protoc -I . \
    --js_out=import_style=commonjs:${CLI_SRC} \
    --grpc-web_out=import_style=typescript,mode=grpcwebtext:${CLI_SRC} \
     $([ "$d" == "$API_DIR_IN" ] && printf -- '--swagger_out=logtostderr=true:%s' $CLI_SRC) \
    "${f}"
  done

  # Add eslint-disable on top of every generated file
  for f in $(find "$CLI_SRC/$d" -name '*.js'); do
      echo '/* eslint-disable */' | cat - "${f}" > temp && mv temp "${f}"
  done
done

echo -e '\n✨✨ Done! ✨✨\n'