# openilink-hub AGENTS 规则

## Git 推送规则（强制）

1. 默认推送目标分支是 `product`。
2. 推送命令统一使用：
   - `git push origin HEAD:product`
3. 禁止将代码直接推送到其他分支（如 `main`、`dev`、`feat/*`）。
4. 仓库已配置本地 `pre-push` 守卫：非 `product` 目标会被拦截。
5. 仅在极少数特殊场景下，才允许临时绕过：
   - `ALLOW_NON_PRODUCT_PUSH=1 git push ...`
6. 非明确授权场景禁止使用强推（`--force` / `--force-with-lease`）。

## 执行建议

1. 推送前先确认当前提交范围准确（不要混入无关改动）。
2. 推送后立即执行目标环境健康检查与关键链路冒烟。
