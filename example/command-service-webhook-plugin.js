// ==WebhookPlugin==
// @name         Command Service Bridge
// @namespace    github.com/openilink
// @version      1.0.0
// @description  Forward WeChat text messages to an HTTP command service as {"command": "..."}
// @author       Awsl
// @license      MIT
// @homepage     https://github.com/openilink/openilink-hub
// @icon         🛰️
// @match        text
// @connect      bhwa233-api.vercel.app
// @grant        skip
// ==/WebhookPlugin==

function onRequest(ctx) {
  var command = (ctx.msg && ctx.msg.content) ? String(ctx.msg.content) : "";
  command = command.replace(/^\s+|\s+$/g, "");

  if (!command) {
    skip();
    return;
  }

  ctx.req.url = "https://bhwa233-api.vercel.app/api/command";
  ctx.req.method = "POST";
  ctx.req.headers = ctx.req.headers || {};
  ctx.req.headers["accept"] = "application/json";
  ctx.req.headers["Content-Type"] = "application/json";
  ctx.req.body = JSON.stringify({
    command: command
  });
}
