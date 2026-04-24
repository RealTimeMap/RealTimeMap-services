FROM node:22-alpine AS builder

WORKDIR /build

# Копируем package files для кэширования зависимостей
COPY services/docs-service/package.json services/docs-service/package-lock.json ./

RUN npm ci

# Копируем исходный код
COPY services/docs-service/ ./

# Сборка продакшн билда с base path для traefik
ENV VITE_BASE=/docs-ui/
RUN npm run build


FROM nginx:1.27-alpine AS runtime

# Удаляем дефолтный конфиг
RUN rm /etc/nginx/conf.d/default.conf

# Копируем кастомный конфиг nginx
COPY services/docs-service/nginx.conf /etc/nginx/conf.d/default.conf

# Копируем собранное SPA
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
