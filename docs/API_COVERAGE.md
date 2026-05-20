# Canvas API coverage
## as of raycanvas v0.2.4

| Canvas API | raycanvas | Notes |
|---|---|---|
| `fillStyle` | ✓ | CSS color strings, cached |
| `strokeStyle` | ✓ | CSS color strings, cached |
| `lineWidth` | ✓ | |
| `globalAlpha` | ✓ | |
| `font` | ✓ | CSS font strings, baked atlas per size |
| `textAlign` | ✓ | left / center / right |
| `textBaseline` | ✓ | alphabetic / top / middle / bottom |
| `lineCap` | ✓ | butt / round / square |
| `lineJoin` | ✓ | miter / round / bevel |
| `lineDashOffset` | ✓ | |
| `shadowColor` | ✓ | |
| `shadowBlur` | ✓ | gg gaussian blur, cached |
| `shadowOffsetX` | ✓ | |
| `shadowOffsetY` | ✓ | |
| `imageSmoothingEnabled` | ✓ | |
| `direction` | ✓ | ltr / rtl |
| `globalCompositeOperation` | ✗ | not implemented |
| `filter` | ✗ | not implemented |
| `fillRect()` | ✓ | |
| `strokeRect()` | ✓ | |
| `clearRect()` | ✓ | |
| `fillText()` | ✓ | |
| `strokeText()` | ✗ | not implemented |
| `measureText()` | ✓ | returns width only |
| `beginPath()` | ✓ | |
| `closePath()` | ✓ | |
| `moveTo()` | ✓ | |
| `lineTo()` | ✓ | |
| `arc()` | ✓ | |
| `arcTo()` | ✓ | |
| `bezierCurveTo()` | ✓ | anti-aliased via gg, cached |
| `quadraticCurveTo()` | ✗ | not implemented |
| `rect()` | ✓ | |
| `roundRect()` | ✓ | |
| `ellipse()` | ✗ | not implemented |
| `fill()` | ✓ | convex / star-convex only |
| `stroke()` | ✓ | |
| `clip()` | ✓ | rect: scissor; roundRect: corner overdraw |
| `isPointInPath()` | ✗ | not implemented |
| `isPointInStroke()` | ✗ | not implemented |
| `save()` | ✓ | |
| `restore()` | ✓ | |
| `translate()` | ✓ | |
| `scale()` | ✓ | |
| `rotate()` | ✗ | not implemented |
| `transform()` | ✓ | |
| `setTransform()` | ✓ | |
| `resetTransform()` | ✓ | |
| `getTransform()` | ✗ | not implemented |
| `drawImage()` | ✓ | 3-arg, 5-arg, 9-arg forms |
| `createImageData()` | ✗ | not implemented |
| `getImageData()` | ✗ | not implemented |
| `putImageData()` | ✗ | not implemented |
| `createLinearGradient()` | ✗ | not implemented |
| `createRadialGradient()` | ✗ | not implemented |
| `createConicGradient()` | ✗ | not implemented |
| `createPattern()` | ✗ | not implemented |
| `setLineDash()` | ✓ | |
| `getLineDash()` | ✗ | not implemented |

**Additional raycanvas primitives** not in the canvas API but useful for GPU dispatch:
`FillRoundRect`, `StrokeRoundRect`, `FillRoundRectTop`, `StrokeRoundRectTop`, `FillCircle`, `StrokeCircle`, `DrawIcon`, `RegisterIcon`, `RegisterFont`.
