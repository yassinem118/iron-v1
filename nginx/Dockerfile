FROM node:22-alpine AS admin-build
WORKDIR /src
COPY admin/package.json ./
RUN npm install --no-audit --no-fund
COPY admin/ ./
ARG VITE_API_URL=.
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=admin-build /src/dist/ /usr/share/nginx/html/
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
ENV NGINX_ENVSUBST_FILTER=^DOMAIN$
EXPOSE 80 443
