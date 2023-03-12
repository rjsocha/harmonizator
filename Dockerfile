# syntax=docker/dockerfile:1.5
FROM scratch
COPY harmonizator /harmonizator
ENTRYPOINT [ "/harmonizator" ]
