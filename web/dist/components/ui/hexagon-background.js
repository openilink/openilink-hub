import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { cn } from "@/lib/utils";
export function HexagonBackground({
  className,
  hexagonSize = 75,
  hexagonMargin = 3,
  hexagonProps,
  ...props
}) {
  const viewWidth = 1600;
  const viewHeight = 1100;
  const width = hexagonSize;
  const height = Math.sqrt(3) * 0.5 * width;
  const horizontalStep = width + hexagonMargin;
  const verticalStep = height * 0.75 + hexagonMargin;
  const columns = Math.ceil(viewWidth / horizontalStep) + 2;
  const rows = Math.ceil(viewHeight / verticalStep) + 2;
  const points = [
    `${width * 0.25},0`,
    `${width * 0.75},0`,
    `${width},${height / 2}`,
    `${width * 0.75},${height}`,
    `${width * 0.25},${height}`,
    `0,${height / 2}`,
  ].join(" ");
  const hexagons = Array.from({ length: rows * columns }, (_, index) => {
    const row = Math.floor(index / columns);
    const column = index % columns;
    const x = column * horizontalStep + (row % 2 === 0 ? 0 : width / 2);
    const y = row * verticalStep;
    const isAccent = (row + column) % 7 === 0 || (row * column) % 13 === 0;
    return {
      key: `${row}-${column}`,
      x,
      y,
      accent: isAccent,
      delay: ((row + column) % 6) * 0.6,
    };
  });
  return _jsxs("div", {
    className: cn(
      "pointer-events-none absolute inset-0 overflow-hidden [mask-image:radial-gradient(circle_at_center,black_38%,transparent_86%)]",
      className,
    ),
    "aria-hidden": "true",
    ...props,
    children: [
      _jsx("div", {
        className:
          "absolute inset-0 bg-[radial-gradient(circle_at_top,rgba(255,255,255,0.14),transparent_34%),radial-gradient(circle_at_bottom,rgba(255,255,255,0.08),transparent_38%)]",
      }),
      _jsx("svg", {
        viewBox: `0 0 ${viewWidth} ${viewHeight}`,
        className:
          "hexagon-background absolute left-1/2 top-1/2 h-[120%] w-[120%] min-w-[960px] -translate-x-1/2 -translate-y-1/2 opacity-55",
        fill: "none",
        preserveAspectRatio: "xMidYMid slice",
        children: _jsx("g", {
          children: hexagons.map((hexagon) =>
            _jsx(
              "polygon",
              {
                points: points,
                transform: `translate(${hexagon.x} ${hexagon.y})`,
                className: hexagon.accent ? "hexagon-accent" : "hexagon-cell",
                style: { animationDelay: `${hexagon.delay}s` },
                ...hexagonProps,
              },
              hexagon.key,
            ),
          ),
        }),
      }),
    ],
  });
}
