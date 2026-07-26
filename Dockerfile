FROM node:22-bookworm-slim AS base

WORKDIR /usr/src/app

FROM base AS dependencies

COPY package*.json ./
RUN npm ci

FROM dependencies AS build

COPY nest-cli.json tsconfig*.json ./
COPY src ./src
RUN npm run build

FROM base AS production

ENV NODE_ENV=production
ENV PORT=8080

COPY package*.json ./
RUN npm ci --omit=dev && npm cache clean --force

COPY --from=build /usr/src/app/dist ./dist

USER node

EXPOSE 8080

CMD ["node", "dist/main.js"]
