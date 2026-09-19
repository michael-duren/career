FROM debian:bookworm-slim AS build
ARG WHISPER_REF=v1.7.6
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates cmake build-essential && rm -rf /var/lib/apt/lists/*
RUN git clone --depth 1 --branch "$WHISPER_REF" https://github.com/ggml-org/whisper.cpp.git /src
WORKDIR /src
RUN cmake -B build -DGGML_NATIVE=OFF -DWHISPER_BUILD_SERVER=ON -DBUILD_SHARED_LIBS=OFF && cmake --build build --config Release --target whisper-server -j2
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ffmpeg libgomp1 ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /src/build/bin/whisper-server /usr/local/bin/whisper-server
USER 65532:65532
WORKDIR /tmp
ENTRYPOINT ["whisper-server"]
CMD ["--host", "0.0.0.0", "--port", "8080", "--model", "/models/ggml-base.en.bin", "--convert", "--threads", "2"]
