# syntax=docker/dockerfile:1

# Built from the repository root (see docker-compose.yml), so the image can include the API
# spec from docs/api as well as the frontend.

FROM node:24-alpine AS build
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Static files, plus a reverse proxy so the browser talks to one origin (no CORS).
FROM nginxinc/nginx-unprivileged:1.28-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

# API reference at /docs: the Swagger UI bundle with its licence files, the page that starts it,
# and the spec itself, copied from docs/api so there is one source of truth.
ARG SWAGGER=/app/node_modules/swagger-ui-dist
COPY --from=build ${SWAGGER}/swagger-ui-bundle.js ${SWAGGER}/swagger-ui.css ${SWAGGER}/swagger-ui-bundle.js.LICENSE.txt ${SWAGGER}/LICENSE ${SWAGGER}/NOTICE /usr/share/nginx/html/docs/
COPY docs/api/viewer/ /usr/share/nginx/html/docs/
COPY docs/api/openapi.yaml /usr/share/nginx/html/docs/openapi.yaml
EXPOSE 8080
