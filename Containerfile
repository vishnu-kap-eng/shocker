FROM cgr.dev/chainguard/go:latest AS builder
COPY . /app
RUN cd /app && go build -o shocker .

FROM cgr.dev/chainguard/wolfi-base:latest
COPY --from=builder /app/shocker /home/appuser/
RUN adduser appuser -h /home/appuser -s /bin/sh -D && chown appuser:appuser /home/appuser/shocker
USER appuser
ENTRYPOINT ["/home/appuser/shocker"]