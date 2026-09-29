const BALL_SIZE = 56
const CAN_WIDTH = 88
const LID_HEIGHT = 24
const BODY_HEIGHT = 102

// Same vertex count in every step so the browser can interpolate the clip-path.
const FLAT = '0% 0%, 33% 0%, 66% 0%, 100% 0%, 100% 33%, 100% 66%, 100% 100%, 66% 100%, 33% 100%, 0% 100%, 0% 66%, 0% 33%'
const CREASED = '3% 4%, 30% 9%, 70% 2%, 97% 6%, 92% 35%, 99% 70%, 94% 97%, 64% 90%, 36% 99%, 5% 93%, 9% 62%, 1% 30%'
const BALLED = '22% 12%, 45% 4%, 72% 10%, 90% 22%, 96% 48%, 88% 74%, 74% 92%, 48% 97%, 24% 88%, 8% 70%, 3% 44%, 10% 24%'

const BALL_SVG = `
<svg viewBox="0 0 56 56" width="${BALL_SIZE}" height="${BALL_SIZE}">
  <path d="M14 7 L27 3 L41 6 L51 16 L54 30 L49 44 L38 52 L24 54 L11 47 L4 34 L3 20 Z"
        fill="#fbfbfb" stroke="#c9ced4" stroke-width="1.5" stroke-linejoin="round"/>
  <path d="M14 7 L22 21 L35 17 L41 6 M22 21 L16 35 L4 34 M35 17 L40 31 L54 30 M16 35 L28 40 L40 31 M28 40 L24 54 M40 31 L49 44 M22 21 L28 29 L35 17 M28 29 L28 40"
        fill="none" stroke="#d5d9de" stroke-width="1.2" stroke-linejoin="round"/>
  <path d="M8 18 L20 12 M44 40 L36 47" stroke="#e8ebee" stroke-width="2" stroke-linecap="round"/>
</svg>`

const LID_SVG = `
<svg viewBox="0 0 88 24" width="${CAN_WIDTH}" height="${LID_HEIGHT}">
  <rect x="34" y="2" width="20" height="8" rx="4" fill="none" stroke="#6b737c" stroke-width="3"/>
  <rect x="2" y="9" width="84" height="12" rx="4" fill="#aab2bb" stroke="#6b737c" stroke-width="2"/>
</svg>`

const BODY_SVG = `
<svg viewBox="0 0 88 102" width="${CAN_WIDTH}" height="${BODY_HEIGHT}">
  <path d="M6 2 H82 L74 98 Q73 100 71 100 H17 Q15 100 14 98 Z" fill="#9aa3ad" stroke="#6b737c" stroke-width="2" stroke-linejoin="round"/>
  <path d="M26 14 L29 88 M44 14 V88 M62 14 L59 88" stroke="#7b848e" stroke-width="4" stroke-linecap="round"/>
  <path d="M10 6 L16 94" stroke="#c3c9d0" stroke-width="3" stroke-linecap="round" opacity="0.7"/>
</svg>`

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function el(style, html = '') {
  const node = document.createElement('div')
  node.style.cssText = style
  node.innerHTML = html
  return node
}

function run(node, keyframes, options) {
  return node.animate(keyframes, { fill: 'forwards', ...options }).finished
}

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

// Crumples `paper` into a ball and tosses it into a trash can. Resolves when done.
export async function crumpleIntoTrash(paper) {
  if (!paper?.animate || prefersReducedMotion()) return

  const rect = paper.getBoundingClientRect()
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2

  // Above MUI modals (1300) so the ball and can render over the backdrop.
  const layer = el('position:fixed;inset:0;pointer-events:none;z-index:1350;overflow:hidden')
  layer.setAttribute('aria-hidden', 'true')
  const ballX = el(`position:absolute;left:${cx - BALL_SIZE / 2}px;top:${cy - BALL_SIZE / 2}px;opacity:0`)
  const ballY = el('')
  const ball = el(`width:${BALL_SIZE}px;height:${BALL_SIZE}px;filter:drop-shadow(0 4px 6px rgba(0,0,0,.25))`, BALL_SVG)
  const can = el(`position:absolute;right:max(24px, 10vw);bottom:32px;width:${CAN_WIDTH}px;height:${LID_HEIGHT + BODY_HEIGHT - 2}px;transform-origin:50% 100%`)
  const lid = el(`position:absolute;top:0;left:0;transform-origin:6px 20px;z-index:1`, LID_SVG)
  const body = el(`position:absolute;top:${LID_HEIGHT - 2}px;left:0`, BODY_SVG)

  ballY.append(ball)
  ballX.append(ballY)
  can.append(lid, body)
  layer.append(ballX, can)
  document.body.append(layer)

  try {
    const mouth = body.getBoundingClientRect()
    const dx = mouth.left + CAN_WIDTH / 2 - cx
    const dy = mouth.top + 6 - BALL_SIZE / 2 - cy
    const peak = Math.max(Math.min(0, dy) - 140, BALL_SIZE - cy)
    const rise = Math.sqrt(-peak)
    const apex = rise / (rise + Math.sqrt(dy - peak))

    const endScale = `scale(${BALL_SIZE / rect.width}, ${BALL_SIZE / rect.height})`
    const canEnters = run(can, [
      { transform: 'translateY(180px)', opacity: 0 },
      { transform: 'translateY(0)', opacity: 1 },
    ], { duration: 450, delay: 150, easing: 'cubic-bezier(.34,1.56,.64,1)' })

    await run(paper, [
      { transform: 'none', clipPath: `polygon(${FLAT})`, filter: 'brightness(1)' },
      { transform: 'scale(.9, .85) rotate(-4deg)', clipPath: `polygon(${CREASED})`, filter: 'brightness(.96)', offset: 0.35 },
      { transform: `rotate(28deg) ${endScale}`, clipPath: `polygon(${BALLED})`, filter: 'brightness(.9)' },
    ], { duration: 600, easing: 'cubic-bezier(.55,0,.75,.4)' })

    ball.style.transform = 'rotate(28deg)'
    await Promise.all([
      run(paper, [{ opacity: 1 }, { opacity: 0 }], { duration: 120 }),
      run(ballX, [{ opacity: 0 }, { opacity: 1 }], { duration: 120 }),
      canEnters,
    ])

    const flight = 750
    const lidOpens = wait(flight * 0.35).then(() =>
      run(lid, [{ transform: 'rotate(0)' }, { transform: 'rotate(-105deg)' }], { duration: 220, easing: 'cubic-bezier(.2,.8,.3,1.2)' }))
    await Promise.all([
      run(ballX, [{ transform: 'translateX(0)' }, { transform: `translateX(${dx}px)` }], { duration: flight, easing: 'linear' }),
      run(ballY, [
        { transform: 'translateY(0)', easing: 'cubic-bezier(.2,.6,.35,1)' },
        { transform: `translateY(${peak}px)`, offset: apex, easing: 'cubic-bezier(.65,0,.8,.4)' },
        { transform: `translateY(${dy}px)` },
      ], { duration: flight }),
      run(ball, [{ transform: 'rotate(28deg) scale(1)' }, { transform: 'rotate(620deg) scale(.8)' }], { duration: flight, easing: 'linear' }),
      lidOpens,
    ])

    await run(ballY, [
      { transform: `translateY(${dy}px)` },
      { transform: `translateY(${dy + 70}px)` },
    ], { duration: 180, easing: 'ease-in' })

    await run(lid, [{ transform: 'rotate(-105deg)' }, { transform: 'rotate(0)' }], { duration: 180, easing: 'cubic-bezier(.6,0,.9,.5)' })
    await run(can, [
      { transform: 'rotate(0) scale(1, 1)' },
      { transform: 'rotate(-7deg) scale(1.04, .94)', offset: 0.2 },
      { transform: 'rotate(5deg) scale(.98, 1.03)', offset: 0.45 },
      { transform: 'rotate(-2deg) scale(1, 1)', offset: 0.7 },
      { transform: 'rotate(0) scale(1, 1)' },
    ], { duration: 420, easing: 'ease-out' })
    await run(can, [
      { transform: 'translateY(0)', opacity: 1 },
      { transform: 'translateY(180px)', opacity: 0 },
    ], { duration: 300, delay: 150, easing: 'cubic-bezier(.5,0,.9,.4)' })
  } finally {
    layer.remove()
  }
}
