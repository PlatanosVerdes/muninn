// muninn's page: Lit components, one class per piece. The app element fetches; everything below it
// renders what it is given. Nothing here talks to a bot.
import { LitElement, html, svg, css, nothing } from "./vendor/lit-all.min.js";

const REFRESH_MS = 15000;
const PEOPLE_SHOWN = 8;

const ago = (t) => {
  const d = new Date(t);
  if (isNaN(d) || d.getFullYear() < 2000) return "never";
  const s = Math.max(0, Math.round((Date.now() - d) / 1000));
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.round(s / 60)} min ago`;
  if (s < 86400) return `${Math.round(s / 3600)} h ago`;
  return `${Math.round(s / 86400)} d ago`;
};
const num = (n) => (n ?? 0).toLocaleString();

const base = css`
  :host { display: block; }
  * { box-sizing: border-box; }
  .card { background: var(--surface); border: 1px solid var(--line); border-radius: 14px; padding: 18px; min-width: 0; }
  .muted { color: var(--muted); }
  .dot { width: 9px; height: 9px; border-radius: 50%; display: inline-block; background: var(--muted); flex: none; }
  .dot.ok { background: var(--ok); } .dot.bad { background: var(--bad); }
`;

// --- one bot ----------------------------------------------------------------------------------

class BotCard extends LitElement {
  static properties = { bot: { attribute: false }, everyone: { state: true } };
  static styles = [base, css`
    header { display: flex; align-items: center; gap: 12px; margin-bottom: 14px; }
    .face { position: relative; width: 44px; height: 44px; flex: none; }
    .face img, .face .initial { width: 44px; height: 44px; border-radius: 50%; display: grid; place-items: center; object-fit: cover; background: var(--chip); color: var(--muted); font: 700 18px var(--font-display); }
    .face .dot { position: absolute; right: -1px; bottom: -1px; width: 12px; height: 12px; border: 2px solid var(--surface); }
    h2 { margin: 0; font: 700 20px/1.1 var(--font-display); }
    .handle { font-family: var(--font-data); font-size: 13px; color: var(--muted); }
    .chip { margin-left: auto; font: 500 11px var(--font-data); text-transform: uppercase; letter-spacing: .06em; background: var(--chip); color: var(--muted); border-radius: 6px; padding: 2px 8px; }
    .error { color: var(--bad); margin: 0; font-size: 14px; }
    .figures { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 8px; margin-bottom: 16px; }
    .figures b { display: block; font: 800 26px/1.1 var(--font-display); font-variant-numeric: tabular-nums; }
    .figures span { font-size: 12px; color: var(--muted); }
    h3 { margin: 16px 0 8px; font-size: 12px; letter-spacing: .08em; text-transform: uppercase; color: var(--muted); }
    .traffic { display: flex; align-items: flex-end; gap: 14px; flex-wrap: wrap; }
    svg { flex: 1 1 220px; height: 54px; min-width: 0; }
    svg rect { fill: var(--accent); }
    svg line { stroke: var(--line); }
    .totals { font-size: 13px; color: var(--muted); display: grid; gap: 2px; }
    .totals b { color: var(--ink); font-family: var(--font-data); font-weight: 500; }
    .rejected { color: var(--bad); }
    dl { display: grid; grid-template-columns: auto 1fr; gap: 4px 14px; margin: 0; font-size: 14px; }
    dt { color: var(--muted); } dd { margin: 0; font-family: var(--font-data); }
    ul { list-style: none; margin: 0; padding: 0; font-size: 14px; }
    li { display: grid; grid-template-columns: minmax(0, 1fr) auto auto; gap: 12px; padding: 6px 0; border-top: 1px solid var(--line); align-items: baseline; }
    li:first-child { border-top: 0; }
    .who { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .who small { color: var(--muted); font-family: var(--font-data); margin-left: 6px; }
    .when, .count { font-size: 12px; color: var(--muted); white-space: nowrap; font-family: var(--font-data); }
    button { margin-top: 8px; background: none; border: 0; padding: 0; color: var(--accent); font: 600 13px var(--font-ui); cursor: pointer; }
    button:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; border-radius: 4px; }
  `];

  constructor() { super(); this.everyone = false; this.noPhoto = false; }

  face(title, s, ok) {
    const photo = s?.platform === "telegram" && s.bot && !this.noPhoto;
    return html`<div class="face">
      ${photo ? html`<img src="api/avatar/telegram/${s.bot}" alt="" @error=${() => { this.noPhoto = true; this.requestUpdate(); }}>`
        : html`<span class="initial">${(title || "?")[0].toUpperCase()}</span>`}
      <i class="dot ${ok ? "ok" : "bad"}"></i>
    </div>`;
  }

  bars(perMinute) {
    const max = Math.max(1, ...perMinute);
    const w = 600 / perMinute.length;
    return html`<svg viewBox="0 0 600 54" preserveAspectRatio="none" role="img" aria-label="Messages per minute, last hour">
      <line x1="0" y1="53.5" x2="600" y2="53.5"></line>
      ${perMinute.map((n, i) => n ? svg`<rect x=${i * w + 1} y=${54 - (n / max) * 52} width=${w - 2} height=${(n / max) * 52} rx="1.5"></rect>` : nothing)}
    </svg>`;
  }

  render() {
    const b = this.bot, s = b.stats;
    const title = s?.name || s?.bot || b.container;
    if (!s) {
      return html`<article class="card"><header>${this.face(title, null, false)}<h2>${title}</h2><span class="chip">${b.state}</span></header>
        <p class="error">${b.error || "No answer"}</p></article>`;
    }
    const pm = s.traffic.per_minute || [];
    const lastMinute = pm.length > 1 ? pm[pm.length - 2] : 0;
    const people = s.people || [];
    const shown = this.everyone ? people : people.slice(0, PEOPLE_SHOWN);
    return html`<article class="card">
      <header>${this.face(title, s, true)}<div><h2>${title}</h2><span class="handle">@${s.bot}</span></div><span class="chip">${s.platform}</span></header>
      <div class="figures">
        <div><b>${num(s.users.total)}</b><span>users</span></div>
        <div><b>${num(s.users.active_24h)}</b><span>active today</span></div>
        <div><b>${num(s.users.active_7d)}</b><span>this week</span></div>
        <div><b>${num(s.users.new_24h)}</b><span>new today</span></div>
      </div>
      <h3>Traffic, last hour</h3>
      <div class="traffic">
        ${this.bars(pm)}
        <div class="totals">
          <span><b>${num(lastMinute)}</b> last minute</span>
          <span><b>${num(s.traffic.messages_24h)}</b> in 24 h</span>
          <span class=${s.traffic.rejected_24h ? "rejected" : ""}><b>${num(s.traffic.rejected_24h)}</b> turned away</span>
        </div>
      </div>
      ${s.extra?.length ? html`<h3>${title}</h3><dl>${s.extra.map((e) => html`<dt>${e.label}</dt><dd>${e.value}</dd>`)}</dl>` : nothing}
      <h3>People</h3>
      ${people.length ? html`<ul>${shown.map((p) => html`<li>
          <span class="who">${p.name || p.id}${p.username ? html`<small>@${p.username}</small>` : nothing}</span>
          <span class="count">${num(p.messages)} msg</span>
          <span class="when">${ago(p.last_seen)}</span>
        </li>`)}</ul>
        ${people.length > PEOPLE_SHOWN ? html`<button @click=${() => { this.everyone = !this.everyone; }}>${this.everyone ? "Show fewer" : `Show all ${people.length}`}</button>` : nothing}`
        : html`<p class="muted">Nobody has written yet.</p>`}
    </article>`;
  }
}
customElements.define("bot-card", BotCard);

// --- the page ---------------------------------------------------------------------------------

class MuninnApp extends LitElement {
  static properties = { data: { state: true }, error: { state: true } };
  static styles = [base, css`
    main { max-width: 1180px; margin: 0 auto; padding-block: 28px 48px; padding-left: max(16px, env(safe-area-inset-left, 0px)); padding-right: max(16px, env(safe-area-inset-right, 0px)); }
    .top { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; margin-bottom: 22px; }
    .top img { width: 40px; height: 40px; border-radius: 10px; }
    h1 { margin: 0; font: 800 32px/1 var(--font-display); letter-spacing: -.02em; }
    .where { color: var(--muted); font-size: 15px; }
    .version { margin-left: auto; font: 500 12px var(--font-data); color: var(--muted); background: var(--chip); border-radius: 6px; padding: 3px 8px; }
    .summary { display: flex; flex-wrap: wrap; gap: 8px 22px; margin-bottom: 18px; font-size: 14px; color: var(--muted); }
    .summary b { color: var(--ink); font-family: var(--font-data); font-weight: 500; }
    .summary .bad b { color: var(--bad); }
    .bots { display: grid; gap: 14px; grid-template-columns: repeat(auto-fill, minmax(min(100%, 360px), 1fr)); align-items: start; }
    .error { color: var(--bad); }
    .note { margin-top: 28px; font-size: 13px; color: var(--muted); max-width: 70ch; }
  `];

  connectedCallback() { super.connectedCallback(); this.load(); this.timer = setInterval(() => this.load(), REFRESH_MS); }
  disconnectedCallback() { clearInterval(this.timer); super.disconnectedCallback(); }

  async load() {
    try {
      const r = await fetch("api/bots");
      const body = await r.json();
      if (!r.ok) throw new Error(body.error || `HTTP ${r.status}`);
      this.data = body; this.error = "";
    } catch (e) { this.error = e.message; }
  }

  render() {
    // Answering bots first: a dead one is listed, but it should not push the live ones down.
    const bots = [...(this.data?.bots || [])].sort((a, b) => (b.stats ? 1 : 0) - (a.stats ? 1 : 0));
    const ok = bots.filter((b) => b.stats);
    const sum = (f) => ok.reduce((n, b) => n + f(b.stats), 0);
    const rejected = sum((s) => s.traffic.rejected_24h);
    return html`<main>
      <div class="top">
        <img src="icon.svg" alt="">
        <h1>muninn</h1>
        ${this.data?.name ? html`<span class="where">${this.data.name}</span>` : nothing}
        ${this.data ? html`<span class="version">${this.data.version}</span>` : nothing}
      </div>
      ${this.error ? html`<p class="error">Cannot read the bots: ${this.error}</p>` : nothing}
      ${this.data ? html`<div class="summary">
          <span><b>${ok.length}/${bots.length}</b> bots answering</span>
          <span><b>${num(sum((s) => s.users.total))}</b> users</span>
          <span><b>${num(sum((s) => s.users.active_24h))}</b> active today</span>
          <span><b>${num(sum((s) => s.traffic.messages_24h))}</b> messages in 24 h</span>
          <span class=${rejected ? "bad" : ""}><b>${num(rejected)}</b> turned away</span>
        </div>
        ${bots.length ? html`<div class="bots">${bots.map((b) => html`<bot-card .bot=${b}></bot-card>`)}</div>`
          : html`<p class="muted">No bot carries the muninn.stats label yet.</p>`}` : nothing}
      <p class="note">Each bot reports its own figures at /bot-stats; muninn only reads them. Traffic counts start over when a bot restarts; people are kept. Refreshes every 15 s.</p>
    </main>`;
  }
}
customElements.define("muninn-app", MuninnApp);
