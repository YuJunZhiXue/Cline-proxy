package app

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cline 代理管理面板</title>
<style>
:root{
  --bg:#0d0f12;--bg2:#131519;--bg3:#1a1d23;--panel:rgba(255,255,255,.028);
  --border:rgba(255,255,255,.08);--border-strong:rgba(255,255,255,.16);
  --text:#d6dae2;--text2:#8b919d;--text3:#5d626c;
  --accent:#8ab4f8;--accent2:#7ee2a8;--amber:#d9b04a;--danger:#e0705f;
  --accent-grad:linear-gradient(135deg,#8ab4f8,#a8c7fa);
  --glow:0 0 0 1px rgba(138,180,248,.2);
  --status-active-bg:rgba(126,226,168,.1);--status-cooldown-bg:rgba(217,176,74,.1);
  --status-expired-bg:rgba(224,112,95,.1);
  --btn-primary-bg:#2b4a77;--btn-primary-hover:#34578c;
  --btn-success-bg:#27513c;--btn-success-hover:#2f6249;
  --radius:10px;--radius-sm:7px;
}
[data-theme="light"]{
  --bg:#f6f7f8;--bg2:#ffffff;--bg3:#eef0f2;--panel:#ffffff;
  --border:rgba(15,18,22,.1);--border-strong:rgba(15,18,22,.22);
  --text:#1c1f24;--text2:#5c636e;--text3:#9aa0aa;
  --accent:#33629c;--accent2:#2b7a4b;--amber:#96700f;--danger:#b3442f;
  --accent-grad:linear-gradient(135deg,#33629c,#4a7ab5);
  --glow:0 0 0 1px rgba(51,98,156,.18);
  --status-active-bg:rgba(43,122,75,.1);--status-cooldown-bg:rgba(150,112,15,.1);
  --status-expired-bg:rgba(179,68,47,.08);
  --btn-primary-bg:#33629c;--btn-primary-hover:#2a5284;
  --btn-success-bg:#2b7a4b;--btn-success-hover:#246a40;
}
*{margin:0;padding:0;box-sizing:border-box}
html{-webkit-text-size-adjust:100%}
body{font-family:'Inter','Segoe UI','PingFang SC','Microsoft YaHei',system-ui,sans-serif;background:var(--bg);color:var(--text);font-size:14px;line-height:1.55;min-height:100vh}
.mono,code{font-family:'JetBrains Mono','Cascadia Code','Fira Code',Consolas,monospace;font-size:12px}

/* ===== 布局 ===== */
.layout{display:flex;min-height:100vh}
.sidebar{width:236px;background:var(--panel);backdrop-filter:blur(14px);border-right:1px solid var(--border);padding:18px 10px;flex-shrink:0;display:flex;flex-direction:column;position:sticky;top:0;height:100vh;overflow-y:auto}
.sidebar h1{font-size:15px;font-weight:700;padding:2px 10px 16px;border-bottom:1px solid var(--border);margin-bottom:10px;display:flex;align-items:center;gap:8px;letter-spacing:.02em}
.sidebar h1 .logo{width:28px;height:28px;border-radius:7px;background:var(--btn-primary-bg);display:inline-flex;align-items:center;justify-content:center;color:#fff;box-shadow:var(--glow)}
.sidebar h1 .logo svg{width:15px;height:15px}
.sidebar h1 .brand-name{color:var(--text)}
.sidebar h1 .theme-toggle{margin-left:auto;padding:4px 8px}
.sidebar h1 span{color:var(--accent)}
.nav-item{display:flex;align-items:center;gap:10px;padding:8px 12px;border-radius:8px;cursor:pointer;color:var(--text2);transition:.15s;font-size:13.5px;margin-bottom:1px;position:relative}
.nav-item .nav-ico{width:18px;height:18px;display:inline-flex;align-items:center;justify-content:center;flex-shrink:0}
.nav-item .nav-ico svg{width:16px;height:16px;stroke:currentColor;fill:none;stroke-width:1.7;stroke-linecap:round;stroke-linejoin:round}
.nav-item:hover{color:var(--text);background:rgba(255,255,255,.05)}
.nav-item.active{color:var(--text);background:rgba(255,255,255,.07);font-weight:600}
.nav-item.active::before{content:'';position:absolute;left:-10px;top:22%;bottom:22%;width:2px;border-radius:2px;background:var(--accent)}
.nav-item.active .nav-ico svg{stroke:var(--accent)}
.nav-group{font-size:10.5px;font-weight:600;letter-spacing:.12em;text-transform:uppercase;color:var(--text3);padding:14px 12px 5px;user-select:none}
.sidebar-footer{margin-top:auto;padding:12px 8px 4px;font-size:11.5px;color:var(--text3);border-top:1px solid var(--border)}
.sidebar-footer a{color:var(--accent);text-decoration:none}
.main{flex:1;padding:26px 34px 60px;min-width:0;max-width:1500px;margin:0 auto;width:100%}
h2{font-size:21px;margin-bottom:18px;font-weight:700;letter-spacing:.01em}

/* ===== 卡片 ===== */
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:14px;margin-bottom:26px}
.card{background:var(--panel);border:1px solid var(--border);border-radius:var(--radius);padding:18px;position:relative;overflow:hidden;transition:.2s}
.card:hover{border-color:var(--border-strong)}
.card .num{font-size:30px;font-weight:700;font-variant-numeric:tabular-nums;letter-spacing:-.02em;color:var(--text)}
.card .label{font-size:12px;color:var(--text2);margin-top:5px;display:flex;align-items:center;gap:6px}
.card .label::before{content:'';width:7px;height:7px;border-radius:50%;background:var(--lab-c,var(--accent))}
.card .num.green{color:var(--accent2);--lab-c:var(--accent2)}
.card .num.red{color:var(--danger);--lab-c:var(--danger)}
.card .num.yellow{color:var(--amber);--lab-c:var(--amber)}
.card .num.blue{color:var(--accent);--lab-c:var(--accent)}
.cards .card{animation:rise .45s ease both}
.cards .card:nth-child(1){animation-delay:.02s}
.cards .card:nth-child(2){animation-delay:.08s}
.cards .card:nth-child(3){animation-delay:.14s}
.cards .card:nth-child(4){animation-delay:.2s}
@keyframes rise{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:none}}

/* ===== 区块 ===== */
.section{background:var(--panel);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:22px;overflow:hidden;animation:rise .4s ease both}
.section-title{padding:13px 18px;border-bottom:1px solid var(--border);font-weight:600;font-size:14px;display:flex;align-items:center;gap:8px}
.section-title .sec-ico{width:17px;height:17px;display:inline-flex;align-items:center;justify-content:center;flex-shrink:0}
.section-title .sec-ico svg{width:15px;height:15px;stroke:var(--text2);fill:none;stroke-width:1.7;stroke-linecap:round;stroke-linejoin:round}
.section-body{padding:18px}
.tabs{display:flex;border-bottom:1px solid var(--border);padding:0 8px;gap:4px;overflow-x:auto}
.tab{padding:11px 18px;cursor:pointer;color:var(--text2);border-bottom:2px solid transparent;font-size:13px;white-space:nowrap;transition:.15s}
.tab:hover{color:var(--text);background:rgba(255,255,255,.04)}
.tab.active{color:var(--text);border-bottom-color:var(--accent);font-weight:600}
.tab-content{display:none;padding:18px}
.tab-content.active{display:block;animation:rise .25s ease both}

/* ===== 表格 ===== */
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:10px 14px;border-bottom:1px solid var(--border);font-size:13px;white-space:nowrap}
th{color:var(--text3);font-weight:600;font-size:11.5px;text-transform:uppercase;letter-spacing:.06em}
tbody tr{transition:.12s}
tbody tr:hover{background:rgba(255,255,255,.035)}
tbody tr:last-child td{border-bottom:none}

/* ===== 状态徽章 ===== */
.status{display:inline-flex;align-items:center;gap:6px;padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600}
.status.active{background:var(--status-active-bg);color:var(--accent2)}
.status.cooldown{background:var(--status-cooldown-bg);color:var(--amber)}
.status.expired{background:var(--status-expired-bg);color:var(--danger)}
.status-dot{width:7px;height:7px;border-radius:50%;display:inline-block}
.status-dot.active{background:var(--accent2)}
.status-dot.cooldown{background:var(--amber)}
.status-dot.expired{background:var(--danger)}

/* ===== 按钮 ===== */
.btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:7px 15px;border:1px solid var(--border);border-radius:var(--radius-sm);background:rgba(255,255,255,.04);color:var(--text);cursor:pointer;font-size:13px;transition:.15s;text-decoration:none;font-family:inherit;white-space:nowrap}
.btn:hover{background:rgba(255,255,255,.09);border-color:var(--border-strong)}
.btn:active{transform:none}
.btn svg{width:14px;height:14px;stroke:currentColor;fill:none;stroke-width:1.8;stroke-linecap:round;stroke-linejoin:round;flex-shrink:0}
.btn-primary{background:var(--btn-primary-bg);border-color:transparent;color:#fff;font-weight:600}
.btn-primary:hover{background:var(--btn-primary-hover)}
.btn-success{background:var(--btn-success-bg);border-color:transparent;color:#fff;font-weight:600}
.btn-success:hover{background:var(--btn-success-hover)}
.btn-danger{border-color:rgba(224,112,95,.4);color:var(--danger);background:transparent}
.btn-danger:hover{background:rgba(224,112,95,.1);border-color:var(--danger)}
.btn-sm{padding:3px 10px;font-size:12px;border-radius:6px}
.btn-sm svg{width:13px;height:13px}

/* ===== 表单 ===== */
input,textarea,select{width:100%;padding:9px 13px;background:rgba(0,0,0,.25);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:13px;font-family:inherit;transition:.15s}
[data-theme="light"] input,[data-theme="light"] textarea,[data-theme="light"] select{background:#fff}
input::placeholder,textarea::placeholder{color:var(--text3)}
input:focus,textarea:focus,select:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 3px rgba(138,180,248,.12)}
textarea{resize:vertical;min-height:84px;font-family:'JetBrains Mono','Cascadia Code',Consolas,monospace;font-size:12px}
select{cursor:pointer;appearance:none;background-image:linear-gradient(45deg,transparent 50%,var(--text2) 50%),linear-gradient(135deg,var(--text2) 50%,transparent 50%);background-position:calc(100% - 18px) 55%,calc(100% - 13px) 55%;background-size:5px 5px;background-repeat:no-repeat;padding-right:32px}
.form-row{display:flex;gap:14px;align-items:flex-end;margin-bottom:14px;flex-wrap:wrap}
.form-row .field{flex:1;min-width:180px}
.form-row .field label{display:block;font-size:12px;color:var(--text2);margin-bottom:6px;font-weight:500}
.form-actions{display:flex;gap:10px;margin-top:14px;flex-wrap:wrap}
.flex{display:flex;align-items:center;gap:8px}
.gap-4{gap:4px}
.text-right{text-align:right}
.mt-8{margin-top:8px}
.inline-flex{display:inline-flex;align-items:center;gap:6px}
.justify-between{display:flex;justify-content:space-between;align-items:center;gap:10px;flex-wrap:wrap}
.hint{font-size:12px;color:var(--text2);margin-top:8px;line-height:1.6}
.hint strong{color:var(--text)}

/* ===== 仪表盘：快速开始与端点 ===== */
.steps{display:flex;flex-direction:column;gap:0}
.step{display:flex;gap:14px;position:relative;padding-bottom:18px}
.step:last-child{padding-bottom:0}
.step::before{content:'';position:absolute;left:13px;top:30px;bottom:2px;width:2px;background:var(--border);border-radius:2px}
.step:last-child::before{display:none}
.step-no{width:27px;height:27px;flex-shrink:0;border-radius:50%;border:1px solid var(--border-strong);color:var(--text2);font-weight:600;font-size:12.5px;display:flex;align-items:center;justify-content:center;background:rgba(255,255,255,.03)}
.step-title{font-weight:600;font-size:13.5px;margin-top:3px}
.step-desc{font-size:12.5px;color:var(--text2);margin-top:3px;line-height:1.6}
.step-desc code{background:rgba(255,255,255,.07);padding:1px 7px;border-radius:5px}
.endpoint-list{display:flex;flex-direction:column;gap:8px}
.endpoint{display:flex;align-items:center;gap:12px;padding:9px 13px;background:rgba(255,255,255,.025);border:1px solid var(--border);border-radius:8px;flex-wrap:wrap}
.endpoint code{background:rgba(138,180,248,.08);border:1px solid rgba(138,180,248,.2);color:var(--accent);padding:3px 10px;border-radius:6px;font-size:11.5px;white-space:nowrap}
.endpoint span{font-size:12px;color:var(--text2);flex:1;min-width:160px}
.danger-zone{border-color:rgba(224,112,95,.28)}
.danger-zone .section-title{color:var(--danger)}
.auto-pill{font-size:11px;color:var(--text2);background:rgba(255,255,255,.05);border:1px solid var(--border);border-radius:999px;padding:2px 10px;white-space:nowrap}
.auto-pill.paused{color:var(--text3);border-color:var(--border)}

/* ===== Toast ===== */
.toast{position:fixed;top:22px;right:22px;padding:12px 20px;border-radius:12px;color:#fff;z-index:9999;opacity:0;transform:translateY(-12px) scale(.97);transition:.3s cubic-bezier(.2,.9,.3,1.2);font-size:13px;max-width:420px;backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.14);box-shadow:0 12px 40px rgba(2,6,23,.5);white-space:pre-line}
.toast.show{opacity:1;transform:none}
.toast.success{background:#2b5a44}
.toast.error{background:#7a3a30}
.toast.info{background:#31465e}
.toast.warning{background:#6e5619}

/* ===== 杂项 ===== */
.loading{display:inline-block;width:14px;height:14px;border:2px solid var(--text3);border-top-color:var(--accent);border-radius:50%;animation:spin .7s linear infinite;vertical-align:-2px}
@keyframes spin{to{transform:rotate(360deg)}}
.empty{padding:30px;text-align:center;color:var(--text2)}
.empty-state{padding:44px 20px;text-align:center;color:var(--text2)}
.empty-state .icon{font-size:40px;margin-bottom:10px;display:block;opacity:.8}
.key-display{background:rgba(0,0,0,.3);padding:9px 13px;border-radius:var(--radius-sm);border:1px solid var(--border);font-family:'JetBrains Mono','Cascadia Code',Consolas,monospace;font-size:12px;word-break:break-all;cursor:pointer;transition:.15s}
.key-display:hover{border-color:var(--accent)}
.copy-icon{cursor:pointer;color:var(--text2);padding:2px 6px;border-radius:4px}
.copy-icon:hover{color:var(--text);background:var(--bg3)}
.model-tag{display:inline-block;padding:2px 9px;border-radius:6px;font-size:11px;background:rgba(255,255,255,.06);color:var(--text2);margin:2px;letter-spacing:.02em}
.model-tag.free{border:1px solid rgba(126,226,168,.35);color:var(--accent2);background:rgba(126,226,168,.07)}
.model-tag.pass{border:1px solid rgba(217,176,74,.35);color:var(--amber);background:rgba(217,176,74,.07)}
.theme-toggle{display:inline-flex;align-items:center;gap:5px;padding:4px 9px;border:1px solid var(--border);border-radius:7px;background:rgba(255,255,255,.04);color:var(--text2);cursor:pointer;font-size:12px;transition:.15s;font-family:inherit}
.theme-toggle:hover{color:var(--text);background:rgba(255,255,255,.09)}
[data-theme="dark"] .theme-toggle .light-label{display:none}
body:not([data-theme="dark"]) .theme-toggle .dark-label{display:none}
.probe-pill{font-size:11px;color:var(--text3)}
.oauth-card{border:1px solid var(--border);border-radius:10px;padding:16px;background:rgba(255,255,255,.03);margin-top:14px}
.stat-mini{font-family:'JetBrains Mono',Consolas,monospace;font-size:12px;color:var(--text2)}

/* ===== 响应式 ===== */
@media (max-width:980px){
  .layout{flex-direction:column}
  .sidebar{width:100%;height:auto;position:sticky;top:0;flex-direction:row;align-items:center;padding:10px 12px;overflow-x:auto;gap:4px;z-index:50}
  .sidebar h1{display:flex;align-items:center;border-bottom:none;margin:0;padding:0 6px 0 0;gap:6px;font-size:13px;white-space:nowrap}
  .sidebar h1 .brand-name{display:none}
  .sidebar .nav-item{padding:7px 11px;white-space:nowrap}
  .sidebar .nav-item.active::before{display:none}
  .nav-group{display:none}
  .sidebar .theme-toggle{margin-left:4px}
  .sidebar-footer{display:none}
  .main{padding:20px 16px 48px}
}
@media (max-width:560px){
  .cards{grid-template-columns:repeat(2,1fr)}
  h2{font-size:18px}
  .section-title{padding:11px 14px}
  .section-body{padding:14px}
  .form-row .field{min-width:100%}
}
</style>
</head>
<body>
<div class="layout">
<div class="sidebar">
<h1><span class="logo"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h7l-1 8 10-12h-7l1-8z"/></svg></span><span class="brand-name">Cline Proxy</span><button class="theme-toggle" onclick="toggleTheme()" title="切换主题"><span class="icon" id="themeIcon"></span><span class="light-label">浅色</span><span class="dark-label">深色</span></button></h1>
<div class="nav-item active" data-tab="dashboard"><span class="nav-ico"><svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/></svg></span> 仪表盘</div>
<div class="nav-group">账号池</div>
<div class="nav-item" data-tab="accounts"><span class="nav-ico"><svg viewBox="0 0 24 24"><circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 3.6-6.5 8-6.5s8 2.5 8 6.5"/></svg></span> 账号管理</div>
<div class="nav-item" data-tab="import"><span class="nav-ico"><svg viewBox="0 0 24 24"><path d="M12 3v10m0 0 4-4m-4 4-4-4"/><path d="M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2"/></svg></span> 导入账号</div>
<div class="nav-group">服务</div>
<div class="nav-item" data-tab="settings"><span class="nav-ico"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.87l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.87-.34 1.7 1.7 0 0 0-1 1.55V21a2 2 0 1 1-4 0v-.09a1.7 1.7 0 0 0-1-1.55 1.7 1.7 0 0 0-1.87.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.7 1.7 0 0 0 .34-1.87 1.7 1.7 0 0 0-1.55-1H3a2 2 0 1 1 0-4h.09a1.7 1.7 0 0 0 1.55-1 1.7 1.7 0 0 0-.34-1.87l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.7 1.7 0 0 0 1.87.34h0a1.7 1.7 0 0 0 1-1.55V3a2 2 0 1 1 4 0v.09a1.7 1.7 0 0 0 1 1.55h0a1.7 1.7 0 0 0 1.87-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.7 1.7 0 0 0-.34 1.87v0a1.7 1.7 0 0 0 1.55 1H21a2 2 0 1 1 0 4h-.09a1.7 1.7 0 0 0-1.55 1z"/></svg></span> 代理设置</div>
<div class="nav-item" data-tab="logs"><span class="nav-ico"><svg viewBox="0 0 24 24"><path d="M8 6h13M8 12h13M8 18h13"/><path d="M3 6h.01M3 12h.01M3 18h.01"/></svg></span> 请求日志</div>
<div class="nav-item" data-tab="opencode"><span class="nav-ico"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14.5 14.5 0 0 1 0 18 14.5 14.5 0 0 1 0-18z"/></svg></span> opencode 免费模型</div>
<div class="sidebar-footer">
  <div>管理面板: <a href="/admin/">/admin/</a></div>
  <div>API 地址: <span id="footerApiAddr">http://127.0.0.1:3457</span></div>
</div>
</div>

<div class="main">

<div id="tab-dashboard" class="tab-panel">
<h2>仪表盘</h2>
<div class="cards">
  <div class="card"><div class="num blue" id="statTotal">-</div><div class="label">账号总数</div></div>
  <div class="card"><div class="num green" id="statActive">-</div><div class="label">活跃</div></div>
  <div class="card"><div class="num yellow" id="statCooldown">-</div><div class="label">冷却</div></div>
  <div class="card"><div class="num red" id="statExpired">-</div><div class="label">已过期</div></div>
</div>
<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M13 2 3 14h7l-1 8 10-12h-7l1-8z"/></svg></span> 快捷操作</div>
  <div class="section-body" style="display:flex;gap:10px;flex-wrap:wrap">
    <button class="btn btn-primary" onclick="switchTab('import')">添加账号</button>
    <button class="btn" onclick="refreshAllTokens()">刷新全部 Token</button>
    <button class="btn" onclick="document.getElementById('fileInput').click()">从文件导入</button>
    <input type="file" id="fileInput" accept=".json,.txt" style="display:none" onchange="handleFileImport(event)">
    <button class="btn" onclick="switchTab('settings');generateKey()">生成 API 密钥</button>
  </div>
</div>
<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg></span> 快速开始</div>
  <div class="section-body">
    <div class="steps">
      <div class="step"><div class="step-no">1</div><div class="step-body"><div class="step-title">导入 Cline 账号</div><div class="step-desc">在「导入账号」页通过 OAuth 登录或粘贴 Token，账号是代理的下游额度来源。</div></div></div>
      <div class="step"><div class="step-no">2</div><div class="step-body"><div class="step-title">生成 API 密钥</div><div class="step-desc">在「代理设置 → API 密钥」生成 Key，用于客户端鉴权；未配置任何 Key 时允许无鉴权访问。</div></div></div>
      <div class="step"><div class="step-no">3</div><div class="step-body"><div class="step-title">配置客户端</div><div class="step-desc">Base URL 填 <code id="quickstartApi">http://127.0.0.1:3457/v1</code>，模型选下方可用模型列表中的任意 ID。</div></div></div>
    </div>
  </div>
</div>
<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M9 12h6M12 9v6"/><circle cx="12" cy="12" r="9"/></svg></span> 接入端点</div>
  <div class="section-body">
    <div class="endpoint-list">
      <div class="endpoint"><code>POST /v1/chat/completions</code><span>OpenAI Chat 格式，大多数工具直接可用</span></div>
      <div class="endpoint"><code>POST /v1/messages</code><span>Anthropic Messages 格式，Claude Code / Cline 等客户端</span></div>
      <div class="endpoint"><code>POST /v1/responses</code><span>OpenAI Responses 格式，Cursor 等客户端</span></div>
      <div class="endpoint"><code>GET /v1/models</code><span>模型列表</span></div>
    </div>
  </div>
</div>
</div>

<div id="tab-accounts" class="tab-panel" style="display:none">
  <div class="flex justify-between" style="margin-bottom:16px">
  <h2>账号管理</h2>
  <div style="display:flex;gap:8px">
    <button class="btn btn-sm" onclick="exportAccounts()">导出账号</button>
    <button class="btn btn-primary btn-sm" onclick="switchTab('import')">添加</button>
    <button class="btn btn-sm" onclick="loadAccounts()">刷新</button>
  </div>
</div>
<div class="hint" style="margin:-4px 0 16px;padding:11px 14px;border:1px solid var(--border);border-radius:10px;background:rgba(148,163,184,.05)">
  <strong style="color:var(--text)">Tokens</strong>为本代理本地统计（输入+输出，上游返回 usage 时精确，否则按请求体估算），用于估算离官方限流还有多远；「测试」按钮发起真实探测请求；「重置」按钮会<strong style="color:var(--text)">探测上游限流状态</strong>：若上游仍限流则保持冷却并提示恢复时间，探测通过才解除冷却并重置今日统计。
</div>
<div class="section">
  <div class="section-body" style="padding:6px">
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>邮箱</th><th>状态</th><th title="本代理本地统计，不代表官方免费额度">今日/累计 Tokens</th><th>最后使用</th><th>创建时间</th><th>操作</th></tr>
      </thead>
      <tbody id="accountTableBody">
        <tr><td colspan="6" class="empty">加载中...</td></tr>
      </tbody>
    </table>
    </div>
  </div>
</div>
</div>

<div id="tab-import" class="tab-panel" style="display:none">
<h2>导入账号</h2>
<div class="section">
  <div class="tabs" id="importTabs">
    <div class="tab active" data-tab="oauth">OAuth 浏览器登录</div>
    <div class="tab" data-tab="token">手动输入 Token</div>
    <div class="tab" data-tab="batch">批量导入</div>
  </div>

  <div id="import-oauth" class="tab-content active">
    <p class="hint">通过浏览器完成 OAuth 认证，支持 Google/GitHub/邮箱登录，自动获取 refreshToken。</p>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="startOAuth()" id="oauthBtn">开始 OAuth 登录</button>
    </div>
    <div id="oauthProgress" style="display:none;margin-top:14px" class="oauth-card">
      <div style="display:flex;align-items:center;gap:14px">
        <div class="loading"></div>
        <div>
          <div style="font-weight:600" id="oauthStatus">等待浏览器授权...</div>
          <div class="hint">
            打开 <a href="#" id="oauthUrl" target="_blank" style="color:var(--accent)"></a>
            并输入代码: <strong style="color:var(--accent);font-size:15px;letter-spacing:2px" id="oauthUserCode"></strong>
          </div>
        </div>
      </div>
    </div>
    <div id="oauthResult" style="display:none;margin-top:14px"></div>
  </div>

  <div id="import-token" class="tab-content">
    <p class="hint">输入已有的 Cline refreshToken，系统会自动验证并加入池。</p>
    <div class="form-row">
      <div class="field">
        <label>Refresh Token *</label>
        <input type="text" id="tokenInput" placeholder="粘贴 refreshToken" style="font-family:'JetBrains Mono',Consolas,monospace">
      </div>
      <div class="field">
        <label>邮箱（可选，留空自动生成）</label>
        <input type="text" id="tokenEmail" placeholder="user@example.com">
      </div>
    </div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="addByToken()">添加账号</button>
    </div>
    <div id="tokenResult" style="margin-top:8px"></div>
  </div>

  <div id="import-batch" class="tab-content">
    <p class="hint">批量导入多个账号。支持 JSON 数组或每行一个 token。</p>
    <div class="form-row">
      <div class="field">
        <label>JSON 数组格式：[{"refreshToken":"...","email":"..."}]</label>
        <textarea id="batchInput" placeholder='[{"refreshToken":"xxx","email":"u1@x.com"},{"refreshToken":"yyy","email":"u2@x.com"}]'></textarea>
      </div>
    </div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="batchImport()">导入全部</button>
      <button class="btn" onclick="document.getElementById('fileInput2').click()">选择文件</button>
      <input type="file" id="fileInput2" accept=".json,.txt" style="display:none" onchange="handleFileImport(event)">
    </div>
    <div id="batchResult" style="margin-top:8px"></div>
  </div>
</div>
</div>

<div id="tab-settings" class="tab-panel" style="display:none">
<h2>代理设置</h2>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M21 2l-2 2m-7.6 7.6a5.5 5.5 0 1 1-7.78 7.78 5.5 5.5 0 0 1 7.78-7.78zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg></span> API 密钥管理</div>
  <div class="section-body">
    <p class="hint">生成的密钥可用于客户端访问代理 API（作为 x-api-key 或 Authorization 头）。</p>
    <div class="form-actions" style="margin-bottom:14px">
      <button class="btn btn-success" onclick="generateKey()">生成新密钥</button>
    </div>
    <div id="keysList"></div>
    <div id="keyGenResult" style="margin-top:8px"></div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/><path d="M12 12v3M8.5 13.5v0M15.5 13.5v0"/></svg></span> 可用模型 <span id="modelsProbeInfo" class="probe-pill" style="font-weight:normal"></span></div>
  <div class="section-body">
    <div class="flex" style="margin-bottom:10px;gap:10px">
      <button class="btn btn-sm btn-primary" onclick="refreshModels()">刷新模型</button>
      <span class="hint" style="margin:0">自动同步上游官方免费模型（60 秒），仅显示不消耗额度的模型</span>
    </div>
    <div id="modelsList">加载中...</div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M12 2v4m0 12v4M2 12h4m12 0h4M4.9 4.9l2.8 2.8m8.6 8.6 2.8 2.8m0-14.2-2.8 2.8M7.7 16.3l-2.8 2.8"/></svg></span> 基础配置</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>监听地址</label><input type="text" id="settingAddr" disabled></div>
      <div class="field">
        <label>默认模型</label>
        <div style="display:flex;gap:6px;align-items:center">
          <select id="settingDefModel" style="flex:1;font-family:'JetBrains Mono',Consolas,monospace"></select>
          <button class="btn btn-sm btn-primary" onclick="saveDefaultModel()">保存</button>
        </div>
      </div>
    </div>
    <div class="form-row">
      <div class="field">
        <label>轮询策略</label>
        <select id="settingStrategy" onchange="updateConfig()">
          <option value="round_robin">轮询 (round_robin)</option>
          <option value="fill">填满 (fill)</option>
          <option value="random">随机 (random)</option>
        </select>
      </div>
      <div class="field"><label>引擎版本</label><input type="text" id="settingVersion" disabled></div>
    </div>
    <div class="form-row">
      <div class="field"><label>账号文件</label><input type="text" id="settingPoolPath" disabled></div>
    </div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M4 7h16M4 12h16M4 17h10"/></svg></span> 请求头配置（模拟 Cline CLI 发出）</div>
  <div class="section-body">
    <div class="table-wrap">
    <table>
      <thead><tr><th style="width:220px">请求头</th><th>值</th><th style="width:40px"></th></tr></thead>
      <tbody id="headersTableBody">
        <tr><td colspan="3" class="empty">加载中...</td></tr>
      </tbody>
    </table>
    </div>
    <div class="form-actions">
      <button class="btn btn-sm" onclick="addHeaderRow()">添加请求头</button>
      <button class="btn btn-sm btn-primary" onclick="saveHeaders()">保存请求头</button>
    </div>
    <div class="hint">这些请求头会附加到所有转发给 Cline API 的请求中，以模拟官方客户端行为。</div>
    <div id="headerSaveResult" style="margin-top:8px"></div>
  </div>
</div>

<div class="section danger-zone">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/></svg></span> 危险操作</div>
  <div class="section-body">
    <div style="display:flex;gap:10px;flex-wrap:wrap">
      <button class="btn btn-danger" onclick="deleteAllAccounts()">删除全部账号</button>
      <button class="btn btn-danger" onclick="deleteAllKeys()">删除全部密钥</button>
    </div>
  </div>
</div>
</div>

<div id="tab-logs" class="tab-panel" style="display:none">
<div class="flex justify-between" style="margin-bottom:16px">
  <h2>请求日志 <span class="probe-pill" style="font-weight:normal">最近 500 条，落盘 data/requests.jsonl</span></h2>
  <div style="display:flex;gap:8px;align-items:center">
    <span class="auto-pill" id="logsAutoPill">自动刷新中</span>
    <button class="btn btn-sm" onclick="toggleLogsAuto()" id="logsAutoBtn">暂停</button>
    <button class="btn btn-sm" onclick="loadLogs()">刷新</button>
  </div>
</div>
<div class="section">
  <div class="section-body" style="padding:6px">
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>时间</th><th>来源</th><th>方法</th><th>路径</th><th>模型</th><th>路由</th><th>状态</th><th>耗时</th></tr>
      </thead>
      <tbody id="logsTableBody">
        <tr><td colspan="8" class="empty">加载中...</td></tr>
      </tbody>
    </table>
    </div>
  </div>
</div>
</div>

<div id="tab-opencode" class="tab-panel" style="display:none">
<h2>opencode 免费模型（统一网关）</h2>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14.5 14.5 0 0 1 0 18 14.5 14.5 0 0 1 0-18z"/></svg></span> 上游配置</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>启用 opencode 上游</label>
        <select id="ocEnabled"><option value="true">开启</option><option value="false">关闭</option></select>
      </div>
      <div class="field"><label>API Key</label><input type="text" id="ocKey" placeholder="public"></div>
    </div>
    <div class="form-row">
      <div class="field"><label>Base URL</label><input type="text" id="ocBaseURL" placeholder="https://opencode.ai/zen/v1"></div>
      <div class="field"><label>代理策略</label>
        <select id="ocStrategy"><option value="round_robin">轮询 round_robin</option><option value="random">随机 random</option><option value="fill">固定 fill</option></select>
      </div>
    </div>
    <div class="form-row">
      <div class="field"><label>代理列表</label>
        <textarea id="ocProxies" rows="3" placeholder="每行一个: http://user:pass@host:port 或 socks5://host:port"></textarea>
      </div>
    </div>
    <div class="form-row">
      <div class="field"><label>代理冷却</label><span id="ocCooldownInfo" class="hint" style="margin:0;align-self:center">-</span></div>
    </div>
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">保存配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></span> 限流防御</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>最大并发</label><input type="text" id="ocMaxConc" placeholder="8"></div>
      <div class="field"><label>限流重试</label><input type="text" id="ocRetries" placeholder="3"></div>
    </div>
    <div class="form-row">
      <div class="field"><label>故障转移</label>
        <select id="ocFailover"><option value="true">开启(切 cline 池)</option><option value="false">关闭</option></select>
      </div>
      <div class="field"><label>失败阈值</label><input type="text" id="ocFailoverCount" placeholder="3"></div>
    </div>
    <div class="form-row">
      <div class="field"><label>转移窗口(分钟)</label><input type="text" id="ocFailoverMinutes" placeholder="5"></div>
      <div class="field"><label>当前状态</label><span id="ocFailoverInfo" class="stat-mini" style="align-self:center">-</span></div>
    </div>
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">保存限流配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M4 8V6a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v2M4 8v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8M4 8h16"/><path d="M10 12h4"/></svg></span> 上下文压缩（opencode 官方机制）</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>自动压缩</label>
        <select id="ocCompactAuto"><option value="true">开启</option><option value="false">关闭</option></select>
      </div>
      <div class="field"><label>预留缓冲</label><input type="text" id="ocCompactBuffer" placeholder="20000"></div>
    </div>
    <div class="form-row">
      <div class="field"><label>尾部保留</label><input type="text" id="ocKeepTokens" placeholder="8000"></div>
      <div class="field"><label>摘要模型</label><input type="text" id="ocSummaryModel" placeholder="留空=同请求模型"></div>
    </div>
    <div class="form-row">
      <div class="field"><label>摘要上限</label><input type="text" id="ocMaxSummary" placeholder="4096"></div>
    </div>
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">保存压缩配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2"/></svg></span> opencode 模型列表
    <button class="btn btn-sm" onclick="refreshOcModels()" style="margin-left:auto">手动同步</button>
  </div>
  <div class="section-body" style="padding:6px">
    <div id="ocModelsList" style="padding:12px">加载中...</div>
  </div>
</div>

<div class="section">
  <div class="section-title"><span class="sec-ico"><svg viewBox="0 0 24 24"><path d="M3 3v18h18"/><path d="M7 15l4-4 3 3 5-6"/></svg></span> opencode 统计</div>
  <div class="section-body">
    <div class="table-wrap"><div id="ocStatsBox"></div></div>
    <div id="ocModelStatsBox" style="margin-top:14px"></div>
  </div>
</div>
</div>

</div>
</div>

<div id="toast" class="toast"></div>

<script>
const API = '/admin/api';

// ========== 主题切换 ==========
function getTheme() {
  return localStorage.getItem('theme') || 'dark';
}
function applyTheme(t) {
  if (t === 'dark') {
    document.documentElement.setAttribute('data-theme', 'dark');
    const ic = document.getElementById('themeIcon');
    if (ic) ic.innerHTML = '<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>';
  } else {
    document.documentElement.setAttribute('data-theme', 'light');
    const ic = document.getElementById('themeIcon');
    if (ic) ic.innerHTML = '<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>';
  }
}
function toggleTheme() {
  const cur = getTheme();
  const next = cur === 'dark' ? 'light' : 'dark';
  localStorage.setItem('theme', next);
  applyTheme(next);
}
applyTheme(getTheme());

const _ = id => document.getElementById(id);
const esc = s => { const d=document.createElement('div'); d.textContent=s||''; return d.innerHTML; };
const fmtNum = n => (n || 0).toLocaleString('zh-CN');
const fmtTokens = n => {
  n = n || 0;
  if (n >= 1000000) return (n / 1000000).toFixed(2).replace(/\.?0+$/, '') + 'M';
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'K';
  return String(n);
};
if (window.location.host) {
  _('footerApiAddr').textContent = 'http://' + window.location.host;
  _('quickstartApi').textContent = 'http://' + window.location.host + '/v1';
}

function toast(msg, t, duration) {
  const el = _('toast');
  el.textContent = msg;
  el.style.whiteSpace = 'pre-line';
  el.className = 'toast ' + (t || 'info') + ' show';
  clearTimeout(el._timer);
  el._timer = setTimeout(() => el.classList.remove('show'), duration || 3500);
}

// ========== 导航 ==========
document.querySelectorAll('.nav-item').forEach(el => {
  el.addEventListener('click', () => {
    if (el.classList.contains('active')) return;
    document.querySelectorAll('.nav-item').forEach(e => e.classList.remove('active'));
    el.classList.add('active');
    document.querySelectorAll('.tab-panel').forEach(e => e.style.display = 'none');
    _('tab-' + el.dataset.tab).style.display = 'block';
    if (el.dataset.tab === 'dashboard') { loadStats(); loadAccounts(); }
    if (el.dataset.tab === 'accounts') loadAccounts();
    if (el.dataset.tab === 'settings') { loadKeys(); loadModels(); loadConfig(); }
    if (el.dataset.tab === 'logs') loadLogs();
    if (el.dataset.tab === 'opencode') { loadOcConfig(); loadOcModels(); loadOcStats(); }
  });
});

function switchTab(name) {
  document.querySelectorAll('.nav-item').forEach(e => {
    e.classList.toggle('active', e.dataset.tab === name);
  });
  document.querySelectorAll('.tab-panel').forEach(e => e.style.display = 'none');
  _('tab-' + name).style.display = 'block';
  if (name === 'dashboard') { loadStats(); loadAccounts(); }
  if (name === 'accounts') loadAccounts();
  if (name === 'settings') { loadKeys(); loadModels(); }
  if (name === 'logs') loadLogs();
  if (name === 'opencode') { loadOcConfig(); loadOcModels(); loadOcStats(); }
}

// 导入子标签
document.querySelectorAll('#importTabs .tab').forEach(el => {
  el.addEventListener('click', () => {
    document.querySelectorAll('#importTabs .tab').forEach(e => e.classList.remove('active'));
    el.classList.add('active');
    document.querySelectorAll('#import-oauth,#import-token,#import-batch').forEach(e => e.classList.remove('active'));
    _('import-' + el.dataset.tab).classList.add('active');
  });
});

// ========== API 请求 ==========
async function api(method, path, body) {
  const opts = { method, headers: {} };
  if (body) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(body); }
  const res = await fetch(API + path, opts);
  const data = await res.json();
  if (!data.success && data.error) throw new Error(data.error);
  return data;
}

// ========== 仪表盘 ==========
async function loadStats() {
  try {
    const d = await api('GET', '/stats');
    const s = d.data;
    _('statTotal').textContent = s.total;
    _('statActive').textContent = s.active;
    _('statCooldown').textContent = s.cooldown;
    _('statExpired').textContent = s.expired;
    if (s.version) _('settingVersion').value = s.version;
    if (s.strategy) _('settingStrategy').value = s.strategy;
  } catch (e) { /* ignore */ }
}

// ========== 账号管理 ==========
async function loadAccounts() {
  try {
    const d = await api('GET', '/accounts');
    const list = d.data.accounts;
    const tbody = _('accountTableBody');
    if (!list || list.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty">暂无账号，前往 <a href="#" onclick="switchTab(\'import\')" style="color:var(--accent);cursor:pointer">导入账号</a> 页添加</td></tr>';
      return;
    }
    const sn = { active: '活跃', cooldown: '冷却', expired: '已过期' };
    tbody.innerHTML = list.map(a => {
      const lu = a.lastUsed ? new Date(a.lastUsed).toLocaleString('zh-CN') : '-';
      const cr = a.createdAt ? new Date(a.createdAt).toLocaleString('zh-CN') : '-';
      // 冷却标签：展示预计恢复时间
      let statusExtra = '';
      if (a.status === 'cooldown') {
        const until = a.cooldownUntil ? new Date(a.cooldownUntil).toLocaleString('zh-CN') : '';
        statusExtra = until ? '<div style="font-size:10px;color:var(--text3);margin-top:2px">预计 ' + esc(until) + ' 恢复</div>' : '';
      }
      return '<tr>' +
        '<td>' + esc(a.email) + '</td>' +
        '<td><span class="status ' + a.status + '"><span class="status-dot ' + a.status + '"></span>' + (sn[a.status] || a.status) + '</span>' + statusExtra + '</td>' +
          '<td title="今日 ' + fmtNum(a.tokensToday) + ' / 累计 ' + fmtNum(a.tokensTotal) + ' tokens（上游返回 usage 时精确，否则为估算值）">' + fmtTokens(a.tokensToday) + ' / ' + fmtTokens(a.tokensTotal) + '</td>' +
        '<td class="mono" style="font-size:11px">' + lu + '</td>' +
        '<td class="mono" style="font-size:11px">' + cr + '</td>' +
        '<td style="white-space:nowrap">' +
          '<button class="btn btn-sm" onclick="testAccount(\'' + a.accountId + '\', this)" title="测试账号是否可用（成功会清除冷却/过期状态）">测试</button> ' +
          '<button class="btn btn-sm" onclick="resetAccount(\'' + a.accountId + '\', this)" title="检测限流并解除：探测上游，若仍限流则保持冷却并提示恢复时间">重置</button> ' +
          '<button class="btn btn-sm btn-danger" onclick="deleteAccount(\'' + a.accountId + '\')" title="删除">删除</button>' +
        '</td></tr>';
    }).join('');
  } catch (e) { toast('加载账号失败: ' + e.message, 'error'); }
}

async function testAccount(id, btn) {
  const original = btn ? btn.innerHTML : '';
  if (btn) { btn.disabled = true; btn.innerHTML = '<span class="loading"></span>测试中'; }
  try {
    const d = await api('POST', '/accounts/test', { accountId: id });
    const r = d.data || {};
    const statusMap = { active: '可用', cooldown: '冷却', expired: '已失效', error: '错误' };
    const label = statusMap[r.status] || r.status;
    const prevMap = { active: '活跃', cooldown: '冷却', expired: '已过期', '': '' };
    let msg = '账号 ' + esc(r.email || '') + ' — ' + label;
    if (r.prevStatus && r.prevStatus !== r.status) msg += '（原状态: ' + (prevMap[r.prevStatus] || r.prevStatus) + '）';
    if (r.cooldownUntil) msg += '\n预计恢复: ' + esc(r.cooldownUntil);
    if (r.remaining) msg += '（剩余 ' + esc(r.remaining) + '）';
    if (r.reason) msg += '\n原因: ' + esc(r.reason);
    if (r.httpStatus) msg += '\nHTTP: ' + r.httpStatus;
    const type = r.status === 'active' ? 'success' : (r.status === 'cooldown' ? 'warning' : 'error');
    toast(msg, type, 6000);
    loadAccounts(); loadStats();
  } catch (e) {
    toast('测试失败: ' + e.message, 'error');
  } finally {
    if (btn) { btn.disabled = false; btn.innerHTML = original; }
  }
}

async function deleteAccount(id) {
  if (!confirm('确定删除此账号？')) return;
  try {
    await api('POST', '/accounts/delete', { accountId: id });
    toast('账号已删除', 'success');
    loadAccounts(); loadStats();
  } catch (e) { toast('删除失败: ' + e.message, 'error'); }
}

async function resetAccount(id, btn) {
  const original = btn ? btn.innerHTML : '';
  if (btn) { btn.disabled = true; btn.innerHTML = '<span class="loading"></span>检测中'; }
  try {
    const d = await api('POST', '/accounts/reset', { accountId: id });
    const r = d.data || {};
    const type = d.success ? 'success' : (r.status === 'cooldown' ? 'warning' : 'error');
    let msg = d.message || '检测完成';
    if (r.remaining && r.status !== 'active') msg += '（剩余 ' + esc(r.remaining) + '）';
    toast(msg, type, 6000);
    loadAccounts(); loadStats();
  } catch (e) {
    toast('检测失败: ' + e.message, 'error');
  } finally {
    if (btn) { btn.disabled = false; btn.innerHTML = original; }
  }
}

async function deleteAllAccounts() {
  if (!confirm('确定删除所有账号？此操作不可撤销！')) return;
  try {
    await api('POST', '/accounts/delete-all', {});
    toast('全部账号已删除', 'success');
    loadAccounts(); loadStats();
  } catch (e) { toast('删除失败: ' + e.message, 'error'); }
}

async function refreshAllTokens() {
  try {
    await api('POST', '/accounts/refresh-all', {});
    toast('全部 Token 已刷新', 'success');
    loadAccounts(); loadStats();
  } catch (e) { toast('刷新失败: ' + e.message, 'error'); }
}

// ========== OAuth 登录 ==========
async function startOAuth() {
  const btn = _('oauthBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="loading"></span> 启动中...';
  _('oauthProgress').style.display = 'block';
  _('oauthResult').style.display = 'none';
  _('oauthStatus').textContent = '正在连接 WorkOS...';
  try {
    const d = await api('POST', '/oauth/start');
    const s = d.data;
    _('oauthStatus').textContent = '请在浏览器中打开链接并输入代码';
    const u = _('oauthUrl');
    u.textContent = s.verificationUri;
    u.href = s.verificationUri;
    _('oauthUserCode').textContent = s.userCode;
    const poll = setInterval(async () => {
      try {
        const r = await api('GET', '/oauth/status?sessionId=' + s.sessionId);
        if (r.data.done) {
          clearInterval(poll);
          btn.disabled = false;
          btn.innerHTML = '开始 OAuth 登录';
          if (r.data.success) {
            _('oauthProgress').style.display = 'none';
            _('oauthResult').innerHTML = '<div style="color:var(--accent2);font-weight:600;font-size:14px">账号添加成功: ' + esc(r.data.email) + '</div>';
            _('oauthResult').style.display = 'block';
            loadAccounts(); loadStats();
            toast('账号添加成功！', 'success');
          } else {
            _('oauthStatus').textContent = '失败: ' + (r.data.error || '未知错误');
            toast('OAuth 失败', 'error');
          }
        }
      } catch(e) {}
    }, 2000);
  } catch (e) {
    btn.disabled = false;
    btn.innerHTML = '开始 OAuth 登录';
    _('oauthStatus').textContent = '错误: ' + e.message;
    toast('OAuth 失败: ' + e.message, 'error');
  }
}

// ========== Token 导入 ==========
async function addByToken() {
  const token = _('tokenInput').value.trim();
  if (!token) { toast('请输入 refreshToken', 'error'); return; }
  const email = _('tokenEmail').value.trim();
  try {
    const d = await api('POST', '/accounts/add', { refreshToken: token, email: email || undefined });
    toast('账号添加成功: ' + (d.data.email || ''), 'success');
    _('tokenInput').value = '';
    _('tokenEmail').value = '';
    loadAccounts(); loadStats();
  } catch (e) { toast('添加失败: ' + e.message, 'error'); }
}

// ========== 批量导入 ==========
async function batchImport() {
  const raw = _('batchInput').value.trim();
  if (!raw) { toast('请输入账号数据', 'error'); return; }
  let tokens;
  try { tokens = JSON.parse(raw); if (!Array.isArray(tokens)) tokens = [tokens]; }
  catch { tokens = raw.split('\n').filter(t => t.trim()).map(t => ({ refreshToken: t.trim() })); }
  try {
    const d = await api('POST', '/batch-import', { tokens });
    toast(d.message || '导入完成', 'success');
    _('batchInput').value = '';
    loadAccounts(); loadStats();
  } catch (e) { toast('导入失败: ' + e.message, 'error'); }
}

async function handleFileImport(event) {
  const file = event.target.files[0];
  if (!file) return;
  const text = await file.text();
  let tokens;
  try { tokens = JSON.parse(text); if (!Array.isArray(tokens)) tokens = [tokens]; }
  catch { tokens = text.split('\n').filter(t => t.trim()).map(t => ({ refreshToken: t.trim() })); }
  try {
    const d = await api('POST', '/batch-import', { tokens });
    toast(d.message || '导入了 ' + tokens.length + ' 个账号', 'success');
    loadAccounts(); loadStats();
  } catch (e) { toast('导入失败: ' + e.message, 'error'); }
  event.target.value = '';
}

// ========== API 密钥管理 ==========
async function loadKeys() {
  try {
    const d = await api('GET', '/keys');
    const keys = d.data.keys;
    const el = _('keysList');
    if (!keys || keys.length === 0) {
      el.innerHTML = '<div class="empty-state"><span class="icon"><svg viewBox="0 0 24 24" width="36" height="36" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.6 7.6a5.5 5.5 0 1 1-7.78 7.78 5.5 5.5 0 0 1 7.78-7.78zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg></span>暂无 API 密钥</div>';
      return;
    }
    el.innerHTML = keys.map(k =>
      '<div class="flex" style="margin-bottom:8px">' +
        '<span class="key-display" style="flex:1" onclick="copyText(\'' + k + '\')" title="点击复制">' + esc(k) + '</span>' +
        '<button class="btn btn-sm btn-danger" onclick="deleteKey(\'' + k + '\')">删除</button>' +
      '</div>'
    ).join('');
  } catch (e) { _('keysList').innerHTML = '<div class="empty">加载失败</div>'; }
}

async function generateKey() {
  try {
    const d = await api('POST', '/keys/generate');
    const key = d.data.key;
    _('keyGenResult').innerHTML =
      '<div style="background:rgba(52,211,153,.08);border:1px solid rgba(52,211,153,.4);border-radius:10px;padding:12px">' +
        '<div style="color:var(--accent2);font-weight:600;margin-bottom:8px">新密钥已生成（点击复制）</div>' +
        '<div class="key-display" onclick="copyText(\'' + key + '\')">' + esc(key) + '</div>' +
      '</div>';
    loadKeys();
    toast('密钥已生成', 'success');
    setTimeout(() => _('keyGenResult').innerHTML = '', 8000);
  } catch (e) { toast('生成失败: ' + e.message, 'error'); }
}

async function deleteKey(key) {
  if (!confirm('确定删除此密钥？')) return;
  try {
    await api('POST', '/keys/delete', { key });
    toast('密钥已删除', 'success');
    loadKeys();
  } catch (e) { toast('删除失败: ' + e.message, 'error'); }
}

async function deleteAllKeys() {
  if (!confirm('确定删除所有 API 密钥？')) return;
  try {
    const d = await api('GET', '/keys');
    const keys = d.data.keys || [];
    for (const k of keys) await api('POST', '/keys/delete', { key: k });
    toast('全部密钥已删除', 'success');
    loadKeys();
  } catch (e) { toast('删除失败: ' + e.message, 'error'); }
}

function copyText(t) {
  navigator.clipboard.writeText(t).then(() => toast('已复制到剪贴板', 'success')).catch(() => {
    const ta = document.createElement('textarea');
    ta.value = t; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta);
    toast('已复制到剪贴板', 'success');
  });
}

// ========== 请求日志 ==========
const ROUTE_LABEL = { zen: 'opencode', cline: 'cline 池', admin: '管理', meta: '元信息', other: '其他' };
const STATUS_CLASS = s => s >= 500 ? 'color:var(--danger)' : (s >= 400 ? 'color:var(--amber)' : 'color:var(--accent2)');
let logsAuto = true;

function toggleLogsAuto() {
  logsAuto = !logsAuto;
  _('logsAutoBtn').textContent = logsAuto ? '暂停' : '恢复';
  const pill = _('logsAutoPill');
  pill.textContent = logsAuto ? '自动刷新中' : '已暂停';
  pill.classList.toggle('paused', !logsAuto);
  if (logsAuto) loadLogs();
}

async function loadLogs() {
  try {
    const d = await api('GET', '/logs');
    const logs = d.data.logs || [];
    const tbody = _('logsTableBody');
    if (!logs.length) { tbody.innerHTML = '<tr><td colspan="8" class="empty">暂无请求记录</td></tr>'; return; }
    tbody.innerHTML = logs.map(l => {
      const t = l.time ? new Date(l.time).toLocaleString('zh-CN') : '-';
      const route = ROUTE_LABEL[l.route] || l.route || '-';
      const st = l.status || 0;
      return '<tr>' +
        '<td class="mono" style="font-size:11px">' + t + '</td>' +
        '<td class="mono" style="font-size:11px">' + esc(l.client || '-') + '</td>' +
        '<td>' + esc(l.method || '-') + '</td>' +
        '<td class="mono" style="font-size:11px">' + esc(l.path || '-') + '</td>' +
        '<td class="mono" style="font-size:12px">' + esc(l.model || '-') + '</td>' +
        '<td><span class="model-tag">' + esc(route) + '</span></td>' +
        '<td style="font-weight:600;color:' + STATUS_CLASS(st) + '">' + st + '</td>' +
        '<td class="mono" style="font-size:11px">' + (l.durationMs != null ? l.durationMs + ' ms' : '-') + '</td>' +
      '</tr>';
    }).join('');
  } catch (e) { tbody.innerHTML = '<tr><td colspan="8" class="empty">加载失败</td></tr>'; }
}

// ========== 导出账号 ==========
async function exportAccounts() {
  try {
    const res = await fetch(API + '/accounts/export');
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = 'cline-accounts-export.json';
    document.body.appendChild(a); a.click(); document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast('账号已导出（JSON）', 'success');
  } catch (e) { toast('导出失败: ' + e.message, 'error'); }
}

// ========== 配置管理 ==========
async function updateConfig() {
  const strategy = _('settingStrategy').value;
  try {
    await api('POST', '/config/update', { strategy });
    toast('策略已更新为: ' + strategy, 'success');
  } catch (e) { toast('更新失败: ' + e.message, 'error'); }
}

function addHeaderRow() {
  const tbody = _('headersTableBody');
  const tr = document.createElement('tr');
  tr.innerHTML =
    '<td><input type="text" class="header-key" placeholder="Header-Name" style="font-size:12px;font-family:monospace"></td>' +
    '<td><input type="text" class="header-val" placeholder="value" style="font-size:12px;font-family:monospace"></td>' +
    '<td><button class="btn btn-sm btn-danger" onclick="this.closest(\'tr\').remove()">删除</button></td>';
  tbody.appendChild(tr);
}

async function saveHeaders() {
  const tbody = _('headersTableBody');
  const rows = tbody.querySelectorAll('tr');
  const headers = {};
  let hasEmpty = false;
  rows.forEach(tr => {
    const keyInput = tr.querySelector('.header-key');
    const valInput = tr.querySelector('.header-val');
    if (keyInput && valInput) {
      const k = keyInput.value.trim();
      const v = valInput.value.trim();
      if (k) { headers[k] = v; }
      else if (v) { hasEmpty = true; }
    }
  });
  if (hasEmpty) { toast('存在有值无键的行，已忽略', 'info'); }
  try {
    const d = await api('POST', '/config/update', { headers });
    toast('请求头已保存', 'success');
    _('headerSaveResult').innerHTML =
      '<div style="color:var(--accent2);font-size:12px">已保存 ' + Object.keys(d.data.headers).length + ' 个请求头</div>';
    setTimeout(() => _('headerSaveResult').innerHTML = '', 5000);
    loadConfig();
  } catch (e) { toast('保存失败: ' + e.message, 'error'); }
}

const MODEL_STYLE = {
  active:  { label: '可用', css: 'color:var(--accent2);border:1px solid rgba(52,211,153,.5);background:rgba(52,211,153,.08)' },
  empty:   { label: '响应为空', css: 'color:var(--amber);border:1px solid rgba(245,158,11,.5);background:rgba(245,158,11,.08)' },
  pass:    { label: '需订阅', css: 'color:var(--amber);border:1px solid rgba(245,158,11,.5);background:rgba(245,158,11,.08)' },
  removed: { label: '已下架', css: 'color:var(--text3);border:1px solid var(--border)' },
  error:   { label: '异常', css: 'color:var(--danger);border:1px solid rgba(248,113,113,.5);background:rgba(248,113,113,.08)' },
  unknown: { label: '未探测', css: 'color:var(--text3);border:1px dashed var(--border-strong)' }
};
const COST_LABEL = { free: '免费', pass: '订阅', quota: '消耗额度' };

async function loadModels() {
  try {
    const d = await api('GET', '/models');
    const models = d.data.models || [];
    let info = '';
    if (d.data.lastSync) info += '· 官方清单: ' + new Date(d.data.lastSync).toLocaleTimeString('zh-CN');
    _('modelsProbeInfo').textContent = info;
    if (!models.length) { _('modelsList').innerHTML = '<div class="empty">暂无模型</div>'; return; }
    _('modelsList').innerHTML = models.map(m => {
      const st = MODEL_STYLE[m.status] || MODEL_STYLE.unknown;
      const cost = COST_LABEL[m.cost] || m.cost || '';
      const synced = m.syncedAt ? new Date(m.syncedAt).toLocaleTimeString('zh-CN') : '-';
      return '<div style="display:flex;align-items:center;gap:10px;padding:8px 12px;margin:5px 0;background:rgba(148,163,184,.06);border:1px solid var(--border);border-radius:10px;transition:.15s">' +
        '<span style="font-family:\'JetBrains Mono\',monospace;font-size:13px;flex:1">' + esc(m.id) + '</span>' +
        (m.cost === 'free' ? '<span style="font-size:11px;color:var(--accent2)">不扣费</span>' : '') +
        (cost ? '<span class="model-tag">' + esc(cost) + '</span>' : '') +
        '<span class="model-tag" style="' + st.css + '">' + st.label + '</span>' +
        '<span style="font-size:11px;color:var(--text3);min-width:60px;text-align:right">' + synced + '</span>' +
        '</div>';
    }).join('');
  } catch (e) { _('modelsList').textContent = '加载失败'; }
}

async function refreshModels() {
  try {
    _('modelsProbeInfo').textContent = '· 同步中...';
    const d = await api('POST', '/models/refresh');
    toast(d.message || '同步已开始', 'info');
    setTimeout(loadModels, 3000);
  } catch (e) { toast('刷新失败: ' + e.message, 'error'); _('modelsProbeInfo').textContent = ''; }
}

async function loadModelOptions() {
  try {
    const d = await api('GET', '/models');
    const models = d.data.models || [];
    const sel = _('settingDefModel');
    if (!sel) return;
    sel.innerHTML = models.map(m => {
      const st = MODEL_STYLE[m.status] || MODEL_STYLE.unknown;
      return '<option value="' + esc(m.id) + '">' + esc(m.id) + ' (' + st.label + ')</option>';
    }).join('');
    const c = await api('GET', '/config');
    if (c.data.defaultModel) sel.value = c.data.defaultModel;
    if (!sel.value && models.length) sel.value = models[0].id;
  } catch (e) { /* ignore */ }
}

async function saveDefaultModel() {
  const v = _('settingDefModel').value;
  if (!v) { toast('请选择模型', 'error'); return; }
  try {
    const d = await api('POST', '/config/update', { defaultModel: v });
    toast('默认模型已保存: ' + d.data.defaultModel, 'success');
  } catch (e) { toast('保存失败: ' + e.message, 'error'); }
}

// ========== 配置加载 ==========
async function loadConfig() {
  try {
    const d = await api('GET', '/config');
    const c = d.data;
    if (c.address) _('settingAddr').value = c.address;
    if (c.strategy) _('settingStrategy').value = c.strategy;
    if (c.version) _('settingVersion').value = c.version;
    if (c.poolPath) _('settingPoolPath').value = c.poolPath;
    loadModelOptions();
    if (c.headers) {
      const tbody = _('headersTableBody');
      tbody.innerHTML = Object.entries(c.headers).map(([k, v]) =>
        '<tr>' +
          '<td><input type="text" class="header-key" value="' + esc(k) + '" style="font-size:12px;font-family:monospace;width:100%"></td>' +
          '<td><input type="text" class="header-val" value="' + esc(v) + '" style="font-size:12px;font-family:monospace;width:100%"></td>' +
          '<td><button class="btn btn-sm btn-danger" onclick="this.closest(\'tr\').remove()">删除</button></td>' +
        '</tr>'
      ).join('');
    }
  } catch (e) { /* ignore */ }
}

// ========== opencode 免费模型 ==========
async function loadOcConfig() {
  try {
    const d = await api('GET', '/opencode/config');
    const c = d.data;
    _('ocEnabled').value = String(c.enabled);
    _('ocKey').value = c.key || 'public';
    _('ocBaseURL').value = c.baseURL || '';
    _('ocProxies').value = (c.proxies || []).join('\n');
    _('ocStrategy').value = c.proxyStrategy || 'round_robin';
    _('ocMaxConc').value = c.maxConcurrency || 8;
    _('ocRetries').value = c.retries || 3;
    _('ocFailover').value = String(c.failover);
    _('ocFailoverCount').value = c.failoverCount || 3;
    _('ocFailoverMinutes').value = c.failoverMinutes || 5;
    _('ocCompactAuto').value = String(c.compaction ? c.compaction.auto : true);
    _('ocCompactBuffer').value = c.compaction ? c.compaction.buffer : 20000;
    _('ocKeepTokens').value = c.compaction ? c.compaction.keepTokens : 8000;
    _('ocSummaryModel').value = c.compaction ? (c.compaction.summaryModel || '') : '';
    _('ocMaxSummary').value = c.compaction ? c.compaction.maxSummary : 4096;
    const rt = c.runtime || {};
    _('ocFailoverInfo').innerHTML = rt.failoverActive
      ? '<span style="color:var(--danger)">故障转移中 (opencode 不可用, 请求走 cline 池)</span>'
      : '<span style="color:var(--accent2)">正常</span>';
    const cd = rt.proxyCooldowns || {};
    const keys = Object.keys(cd);
    _('ocCooldownInfo').textContent = keys.length
      ? keys.map(k => k + ' 冷却至 ' + cd[k]).join('; ')
      : '暂无冷却中的代理';
  } catch (e) { /* ignore */ }
}

async function saveOcConfig() {
  const proxies = _('ocProxies').value.split('\n').map(s => s.trim()).filter(Boolean);
  const PROXY_RE = /^(https?|socks5h?):\/\/[^\s]+:\d+/;
  const bad = proxies.find(p => !PROXY_RE.test(p));
  if (bad) { toast('代理格式无效: ' + bad + '（需 http(s)://host:port 或 socks5://host:port）', 'error'); return; }
  const body = {
    enabled: _('ocEnabled').value === 'true',
    key: _('ocKey').value.trim(),
    baseURL: _('ocBaseURL').value.trim(),
    proxies: proxies,
    proxyStrategy: _('ocStrategy').value,
    maxConcurrency: parseInt(_('ocMaxConc').value) || 8,
    retries: parseInt(_('ocRetries').value) || 3,
    failover: _('ocFailover').value === 'true',
    failoverCount: parseInt(_('ocFailoverCount').value) || 3,
    failoverMinutes: parseInt(_('ocFailoverMinutes').value) || 5,
    compaction: {
      auto: _('ocCompactAuto').value === 'true',
      buffer: parseInt(_('ocCompactBuffer').value) || 20000,
      keepTokens: parseInt(_('ocKeepTokens').value) || 8000,
      summaryModel: _('ocSummaryModel').value.trim(),
      maxSummary: parseInt(_('ocMaxSummary').value) || 4096
    }
  };
  try {
    const d = await api('POST', '/opencode/config/update', body);
    toast('opencode 配置已保存', 'success');
    loadOcConfig();
  } catch (e) { toast('保存失败: ' + e.message, 'error'); }
}

async function loadOcModels() {
  try {
    const d = await api('GET', '/opencode/models');
    const models = d.data.models || [];
    _('ocModelsList').innerHTML = '<div class="table-wrap"><table><thead><tr><th style="text-align:left">模型 ID</th><th>上下文</th><th>输出</th><th>来源</th></tr></thead><tbody>' +
      models.map(m => '<tr><td style="text-align:left;font-family:monospace">' + esc(m.id) + '</td><td>' + m.context + '</td><td>' + m.output + '</td><td>' + m.source + '</td></tr>').join('') +
      '</tbody></table></div><div class="hint">共 ' + models.length + ' 个免费模型（每 10 分钟自动同步）</div>';
  } catch (e) { _('ocModelsList').textContent = '加载失败'; }
}

async function refreshOcModels() {
  try {
    const d = await api('POST', '/opencode/models/refresh');
    toast(d.message || '同步完成', 'success');
    loadOcModels();
  } catch (e) { toast('同步失败: ' + e.message, 'error'); }
}

async function loadOcStats() {
  try {
    const d = await api('GET', '/opencode/stats');
    const t = d.data.today || {}, s = d.data.total || {};
    _('ocStatsBox').innerHTML = '<table><thead><tr><th style="text-align:left"></th><th>请求数</th><th>输入 tokens</th><th>输出 tokens</th><th>压缩消耗</th><th>限流命中</th></tr></thead><tbody>' +
      '<tr><td style="text-align:left">今日</td><td>' + (t.requests || 0) + '</td><td>' + (t.promptTokens || 0) + '</td><td>' + (t.completionTokens || 0) + '</td><td>' + (t.compaction || 0) + '</td><td>' + (t.rateLimited || 0) + '</td></tr>' +
      '<tr><td style="text-align:left">累计</td><td>' + (s.requests || 0) + '</td><td>' + (s.promptTokens || 0) + '</td><td>' + (s.completionTokens || 0) + '</td><td>' + (s.compaction || 0) + '</td><td>' + (s.rateLimited || 0) + '</td></tr>' +
      '</tbody></table>';
    const bm = t.byModel || {};
    const rows = Object.keys(bm).map(k => '<tr><td style="text-align:left;font-family:monospace">' + esc(k) + '</td><td>' + bm[k].requests + '</td><td>' + bm[k].promptTokens + '</td><td>' + bm[k].completionTokens + '</td></tr>').join('');
    _('ocModelStatsBox').innerHTML = '<div style="font-size:13px;font-weight:600;margin-bottom:6px">按模型分布（今日）</div>' +
      '<div class="table-wrap"><table><thead><tr><th style="text-align:left">模型</th><th>请求数</th><th>输入 tokens</th><th>输出 tokens</th></tr></thead><tbody>' +
      (rows || '<tr><td colspan="4" style="text-align:left;color:var(--text2)">暂无数据</td></tr>') + '</tbody></table></div>';
  } catch (e) { /* ignore */ }
}

// ========== 初始化 ==========
loadStats();
loadAccounts();
loadKeys();
loadModels();
loadConfig();
setInterval(() => { loadStats(); }, 10000);
setInterval(() => { loadOcStats(); }, 15000);
setInterval(() => { if (logsAuto && _('tab-logs').style.display !== 'none') loadLogs(); }, 8000);
</script>
</body>
</html>`