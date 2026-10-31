FROM node:22-alpine AS admin-build
WORKDIR /app
COPY admin/package*.json ./
RUN npm ci --no-audit --no-fund
COPY admin/ ./
ARG VITE_API_URL=/
ENV VITE_API_URL=${VITE_API_URL}
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=admin-build /app/dist/ /usr/share/nginx/html/
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
