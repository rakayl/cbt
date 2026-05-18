FROM node:24-alpine AS build
WORKDIR /app
COPY package*.json ./
ENV ELECTRON_SKIP_BINARY_DOWNLOAD=1
RUN if [ -f package-lock.json ]; then npm ci; else npm install; fi
COPY . .
RUN npm run build
FROM nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
