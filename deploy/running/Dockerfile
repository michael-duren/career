FROM debian:bookworm-slim AS build
ARG WHISPER_REF=v1.7.6
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates cmake build-essential && rm -rf /var/lib/apt/lists/*
RUN git clone --depth 1 --branch "$WHISPER_REF" https://github.com/ggml-org/whisper.cpp.git /src
WORKDIR /src
RUN cmake -B build -DGGML_NATIVE=OFF -DWHISPER_BUILD_SERVER=ON -DBUILD_SHARED_LIBS=OFF && cmake --build build --config Release --target whisper-server -j2
# Bake the model into the image so the digest pins code and weights together; fail on checksum drift.
ARG WHISPER_MODEL=base.en
ARG WHISPER_MODEL_SHA256=a03779c86df3323075f5e796cb2ce5029f00ec8869eee3fdfb897afe36c6d002
ADD --chmod=0444 --checksum=sha256:${WHISPER_MODEL_SHA256} https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-${WHISPER_MODEL}.bin /models/ggml-${WHISPER_MODEL}.bin
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg libgomp1 ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/build/bin/whisper-server /usr/local/bin/whisper-server
COPY --from=build /models /models
USER 65532:65532
WORKDIR /tmp
ENTRYPOINT ["whisper-server"]
CMD ["--host", "0.0.0.0", "--port", "8080", "--model", "/models/ggml-base.en.bin", "--convert", "--threads", "2"]
