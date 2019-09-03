FROM node:8 as build-deps
WORKDIR /usr/src/app
COPY app/package.json app/yarn.lock ./
RUN yarn
#COPY . ./
#RUN yarn build

CMD ["yarn", "start"]
