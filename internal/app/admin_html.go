package app

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cline 代理管理面板</title>
<style>
:root{
  --bg:#0b0e17;--bg2:#111527;--bg3:#1a2038;--panel:rgba(148,163,184,.055);
  --border:rgba(148,163,184,.14);--border-strong:rgba(148,163,184,.28);
  --text:#e6edf6;--text2:#8b98b4;--text3:#5b6b89;
  --accent:#22d3ee;--accent2:#34d399;--amber:#f59e0b;--danger:#f87171;
  --accent-grad:linear-gradient(135deg,#22d3ee,#34d399);
  --glow:0 0 0 1px rgba(34,211,238,.25),0 0 24px rgba(34,211,238,.12);
  --status-active-bg:rgba(52,211,153,.14);--status-cooldown-bg:rgba(245,158,11,.14);
  --status-expired-bg:rgba(248,113,113,.14);
  --btn-primary-bg:linear-gradient(135deg,#0ea5e9,#22d3ee);--btn-primary-hover:linear-gradient(135deg,#0284c7,#0ea5e9);
  --btn-success-bg:linear-gradient(135deg,#059669,#34d399);--btn-success-hover:linear-gradient(135deg,#047857,#059669);
  --radius:14px;--radius-sm:9px;
}
[data-theme="light"]{
  --bg:#f3f5fa;--bg2:#ffffff;--bg3:#eef1f7;--panel:rgba(255,255,255,.7);
  --border:rgba(15,23,42,.12);--border-strong:rgba(15,23,42,.26);
  --text:#0f172a;--text2:#57617a;--text3:#8a94ab;
  --accent:#0891b2;--accent2:#059669;--amber:#b45309;--danger:#dc2626;
  --accent-grad:linear-gradient(135deg,#0891b2,#059669);
  --glow:0 0 0 1px rgba(8,145,178,.22),0 6px 24px rgba(8,145,178,.10);
  --status-active-bg:rgba(5,150,105,.12);--status-cooldown-bg:rgba(180,83,9,.12);
  --status-expired-bg:rgba(220,38,38,.10);
  --btn-primary-bg:linear-gradient(135deg,#0284c7,#06b6d4);--btn-primary-hover:linear-gradient(135deg,#0369a1,#0284c7);
  --btn-success-bg:linear-gradient(135deg,#059669,#10b981);--btn-success-hover:linear-gradient(135deg,#047857,#059669);
}
*{margin:0;padding:0;box-sizing:border-box}
html{-webkit-text-size-adjust:100%}
body{font-family:'Inter','Segoe UI','PingFang SC','Microsoft YaHei',system-ui,sans-serif;background:var(--bg);color:var(--text);font-size:14px;line-height:1.55;min-height:100vh}
body::before{content:'';position:fixed;inset:0;z-index:-1;background:
  radial-gradient(900px 500px at 85% -10%,rgba(34,211,238,.09),transparent 60%),
  radial-gradient(800px 500px at -10% 110%,rgba(52,211,153,.07),transparent 60%),
  var(--bg);pointer-events:none}
.mono,code{font-family:'JetBrains Mono','Cascadia Code','Fira Code',Consolas,monospace;font-size:12px}

/* ===== 布局 ===== */
.layout{display:flex;min-height:100vh}
.sidebar{width:236px;background:var(--panel);backdrop-filter:blur(14px);border-right:1px solid var(--border);padding:18px 10px;flex-shrink:0;display:flex;flex-direction:column;position:sticky;top:0;height:100vh;overflow-y:auto}
.sidebar h1{font-size:15px;font-weight:700;padding:2px 10px 16px;border-bottom:1px solid var(--border);margin-bottom:10px;display:flex;align-items:center;gap:8px;letter-spacing:.02em}
.sidebar h1 .logo{width:28px;height:28px;border-radius:8px;background:var(--accent-grad);display:inline-flex;align-items:center;justify-content:center;font-size:14px;color:#04121a;box-shadow:var(--glow)}
.sidebar h1 .brand-name{background:var(--accent-grad);-webkit-background-clip:text;background-clip:text;-webkit-text-fill-color:transparent}
.sidebar h1 .theme-toggle{margin-left:auto;padding:4px 8px}
.sidebar h1 span{color:var(--accent)}
.nav-item{display:flex;align-items:center;gap:10px;padding:9px 12px;border-radius:10px;cursor:pointer;color:var(--text2);transition:.18s;font-size:13.5px;margin-bottom:2px;position:relative}
.nav-item .nav-ico{width:18px;text-align:center;font-size:15px;filter:saturate(.8)}
.nav-item:hover{color:var(--text);background:rgba(148,163,184,.09)}
.nav-item.active{color:var(--text);background:linear-gradient(90deg,rgba(34,211,238,.16),rgba(34,211,238,.05));font-weight:600}
.nav-item.active::before{content:'';position:absolute;left:-10px;top:20%;bottom:20%;width:3px;border-radius:3px;background:var(--accent-grad);box-shadow:0 0 12px rgba(34,211,238,.6)}
.sidebar-footer{margin-top:auto;padding:12px 8px 4px;font-size:11.5px;color:var(--text3);border-top:1px solid var(--border)}
.sidebar-footer a{color:var(--accent);text-decoration:none}
.main{flex:1;padding:26px 34px 60px;min-width:0;max-width:1500px;margin:0 auto;width:100%}
h2{font-size:21px;margin-bottom:18px;font-weight:700;letter-spacing:.01em}

/* ===== 卡片 ===== */
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:14px;margin-bottom:26px}
.card{background:var(--panel);backdrop-filter:blur(10px);border:1px solid var(--border);border-radius:var(--radius);padding:18px;position:relative;overflow:hidden;transition:.2s}
.card::after{content:'';position:absolute;inset:0;background:radial-gradient(220px 80px at 85% -10%,rgba(34,211,238,.12),transparent);pointer-events:none}
.card:hover{transform:translateY(-2px);border-color:var(--border-strong);box-shadow:0 10px 30px rgba(2,6,23,.35)}
.card .num{font-size:30px;font-weight:700;font-variant-numeric:tabular-nums;letter-spacing:-.02em;color:var(--text)}
.card .label{font-size:12px;color:var(--text2);margin-top:5px;display:flex;align-items:center;gap:6px}
.card .label::before{content:'';width:7px;height:7px;border-radius:50%;background:var(--lab-c,var(--accent));box-shadow:0 0 10px var(--lab-c,var(--accent))}
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
.section{background:var(--panel);backdrop-filter:blur(10px);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:22px;overflow:hidden;animation:rise .4s ease both}
.section-title{padding:13px 18px;border-bottom:1px solid var(--border);font-weight:600;font-size:14px;display:flex;align-items:center;gap:8px;background:rgba(148,163,184,.03)}
.section-body{padding:18px}
.tabs{display:flex;border-bottom:1px solid var(--border);padding:0 8px;gap:4px;overflow-x:auto}
.tab{padding:11px 18px;cursor:pointer;color:var(--text2);border-bottom:2px solid transparent;font-size:13px;white-space:nowrap;transition:.15s;border-radius:8px 8px 0 0}
.tab:hover{color:var(--text);background:rgba(148,163,184,.07)}
.tab.active{color:var(--accent);border-bottom-color:var(--accent);font-weight:600}
.tab-content{display:none;padding:18px}
.tab-content.active{display:block;animation:rise .25s ease both}

/* ===== 表格 ===== */
.table-wrap{overflow-x:auto}
table{width:100%;border-collapse:collapse}
th,td{text-align:left;padding:10px 14px;border-bottom:1px solid var(--border);font-size:13px;white-space:nowrap}
th{color:var(--text2);font-weight:600;font-size:11.5px;text-transform:uppercase;letter-spacing:.06em}
tbody tr{transition:.12s}
tbody tr:hover{background:rgba(148,163,184,.06)}
tbody tr:last-child td{border-bottom:none}

/* ===== 状态徽章 ===== */
.status{display:inline-flex;align-items:center;gap:6px;padding:3px 10px;border-radius:999px;font-size:12px;font-weight:600}
.status.active{background:var(--status-active-bg);color:var(--accent2)}
.status.cooldown{background:var(--status-cooldown-bg);color:var(--amber)}
.status.expired{background:var(--status-expired-bg);color:var(--danger)}
.status-dot{width:7px;height:7px;border-radius:50%;display:inline-block}
.status-dot.active{background:var(--accent2);box-shadow:0 0 8px var(--accent2)}
.status-dot.cooldown{background:var(--amber);box-shadow:0 0 8px var(--amber)}
.status-dot.expired{background:var(--danger);box-shadow:0 0 8px var(--danger)}

/* ===== 按钮 ===== */
.btn{display:inline-flex;align-items:center;justify-content:center;gap:6px;padding:7px 15px;border:1px solid var(--border);border-radius:var(--radius-sm);background:rgba(148,163,184,.08);color:var(--text);cursor:pointer;font-size:13px;transition:.18s;text-decoration:none;font-family:inherit;white-space:nowrap}
.btn:hover{background:rgba(148,163,184,.16);border-color:var(--border-strong);transform:translateY(-1px)}
.btn:active{transform:none}
.btn-primary{background:var(--btn-primary-bg);border-color:transparent;color:#04121a;font-weight:600;box-shadow:0 4px 14px rgba(14,165,233,.28)}
.btn-primary:hover{background:var(--btn-primary-hover);box-shadow:0 6px 18px rgba(14,165,233,.38)}
.btn-success{background:var(--btn-success-bg);border-color:transparent;color:#04231a;font-weight:600;box-shadow:0 4px 14px rgba(5,150,105,.28)}
.btn-success:hover{background:var(--btn-success-hover)}
.btn-danger{border-color:rgba(248,113,113,.4);color:var(--danger);background:transparent}
.btn-danger:hover{background:rgba(248,113,113,.12);border-color:var(--danger)}
.btn-sm{padding:3px 10px;font-size:12px;border-radius:7px}
/* 状态徽章 */
.badge{display:inline-block;padding:2px 9px;border-radius:999px;font-size:11px;font-weight:600}
.badge-ok{background:rgba(5,150,105,.15);color:#10b981}
.badge-err{background:rgba(248,113,113,.15);color:#f87171}

/* ===== 表单 ===== */
input,textarea,select{width:100%;padding:9px 13px;background:rgba(2,6,23,.4);border:1px solid var(--border);border-radius:var(--radius-sm);color:var(--text);font-size:13px;font-family:inherit;transition:.15s}
[data-theme="light"] input,[data-theme="light"] textarea,[data-theme="light"] select{background:rgba(15,23,42,.03)}
input::placeholder,textarea::placeholder{color:var(--text3)}
input:focus,textarea:focus,select:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 3px rgba(34,211,238,.15)}
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

/* ===== Toast ===== */
.toast{position:fixed;top:22px;right:22px;padding:12px 20px;border-radius:12px;color:#fff;z-index:9999;opacity:0;transform:translateY(-12px) scale(.97);transition:.3s cubic-bezier(.2,.9,.3,1.2);font-size:13px;max-width:420px;backdrop-filter:blur(12px);border:1px solid rgba(255,255,255,.14);box-shadow:0 12px 40px rgba(2,6,23,.5);white-space:pre-line}
.toast.show{opacity:1;transform:none}
.toast.success{background:rgba(5,150,105,.92)}
.toast.error{background:rgba(220,38,38,.92)}
.toast.info{background:rgba(14,165,233,.92)}
.toast.warning{background:rgba(180,83,9,.92)}

/* ===== 杂项 ===== */
.loading{display:inline-block;width:14px;height:14px;border:2px solid var(--text3);border-top-color:var(--accent);border-radius:50%;animation:spin .7s linear infinite;vertical-align:-2px}
@keyframes spin{to{transform:rotate(360deg)}}
.empty{padding:30px;text-align:center;color:var(--text2)}
.empty-state{padding:44px 20px;text-align:center;color:var(--text2)}
.empty-state .icon{font-size:40px;margin-bottom:10px;display:block;opacity:.8}
.key-display{background:rgba(2,6,23,.45);padding:9px 13px;border-radius:var(--radius-sm);border:1px solid var(--border);font-family:'JetBrains Mono','Cascadia Code',Consolas,monospace;font-size:12px;word-break:break-all;cursor:pointer;transition:.15s}
.key-display:hover{background:rgba(34,211,238,.08);border-color:var(--accent)}
.copy-icon{cursor:pointer;color:var(--text2);padding:2px 6px;border-radius:4px}
.copy-icon:hover{color:var(--text);background:var(--bg3)}
.model-tag{display:inline-block;padding:2px 9px;border-radius:6px;font-size:11px;background:rgba(148,163,184,.1);color:var(--text2);margin:2px;letter-spacing:.02em}
.model-tag.free{border:1px solid rgba(52,211,153,.5);color:var(--accent2);background:rgba(52,211,153,.08)}
.model-tag.pass{border:1px solid rgba(245,158,11,.5);color:var(--amber);background:rgba(245,158,11,.08)}
.theme-toggle{display:inline-flex;align-items:center;gap:5px;padding:4px 9px;border:1px solid var(--border);border-radius:8px;background:rgba(148,163,184,.08);color:var(--text2);cursor:pointer;font-size:12px;transition:.15s;font-family:inherit}
.theme-toggle:hover{color:var(--text);background:rgba(148,163,184,.16)}
[data-theme="dark"] .theme-toggle .light-label{display:none}
body:not([data-theme="dark"]) .theme-toggle .dark-label{display:none}
.probe-pill{font-size:11px;color:var(--text3)}
.oauth-card{border:1px solid var(--border);border-radius:12px;padding:16px;background:rgba(148,163,184,.05);margin-top:14px}
.stat-mini{font-family:'JetBrains Mono',Consolas,monospace;font-size:12px;color:var(--text2)}

/* ===== 响应式 ===== */
@media (max-width:980px){
  .layout{flex-direction:column}
  .sidebar{width:100%;height:auto;position:sticky;top:0;flex-direction:row;align-items:center;padding:10px 12px;overflow-x:auto;gap:4px;z-index:50}
  .sidebar h1{display:flex;align-items:center;border-bottom:none;margin:0;padding:0 6px 0 0;gap:6px;font-size:13px;white-space:nowrap}
  .sidebar h1 .brand-name{display:none}
  .sidebar .nav-item{padding:7px 11px;white-space:nowrap}
  .sidebar .nav-item.active::before{display:none}
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
<h1><span class="logo">⚡</span><span class="brand-name">Cline 代理</span><button class="theme-toggle" onclick="toggleTheme()" title="切换主题"><span class="icon" id="themeIcon">🌙</span><span class="light-label">浅色</span><span class="dark-label">深色</span></button></h1>
<div class="nav-item active" data-tab="dashboard"><span class="nav-ico">📊</span> 仪表盘</div>
<div class="nav-item" data-tab="accounts"><span class="nav-ico">👤</span> 账号管理</div>
<div class="nav-item" data-tab="import"><span class="nav-ico">📥</span> 导入账号</div>
<div class="nav-item" data-tab="settings"><span class="nav-ico">⚙️</span> 设置</div>
<div class="nav-item" data-tab="logs"><span class="nav-ico">📜</span> 请求日志</div>
<div class="nav-item" data-tab="opencode"><span class="nav-ico">🌐</span> opencode 免费模型</div>
<div class="nav-item" data-tab="clinepass"><span class="nav-ico">🧭</span> ClinePass Provider</div>
<div class="nav-item" data-tab="codex"><span class="nav-ico">🚀</span> Codex 上游</div>
<div class="sidebar-footer">
  <div>管理面板: <a href="/admin/">/admin/</a></div>
  <div>API 地址: <span id="footerApiAddr">http://127.0.0.1:3457</span></div>
</div>
</div>

<div class="main">

<div id="tab-dashboard" class="tab-panel">
<h2>📊 仪表盘</h2>
<div class="cards">
  <div class="card"><div class="num blue" id="statTotal">-</div><div class="label">账号总数</div></div>
  <div class="card"><div class="num green" id="statActive">-</div><div class="label">活跃</div></div>
  <div class="card"><div class="num yellow" id="statCooldown">-</div><div class="label">冷却</div></div>
  <div class="card"><div class="num red" id="statExpired">-</div><div class="label">已过期</div></div>
</div>
<div class="section">
  <div class="section-title">📋 快捷操作</div>
  <div class="section-body" style="display:flex;gap:10px;flex-wrap:wrap">
    <button class="btn btn-primary" onclick="switchTab('import')">➕ 添加账号</button>
    <button class="btn" onclick="refreshAllTokens()">🔄 刷新全部 Token</button>
    <button class="btn" onclick="document.getElementById('fileInput').click()">📄 从文件导入</button>
    <input type="file" id="fileInput" accept=".json,.txt" style="display:none" onchange="handleFileImport(event)">
    <button class="btn" onclick="switchTab('settings');generateKey()">🔑 生成 API 密钥</button>
  </div>
</div>
</div>

<div id="tab-accounts" class="tab-panel" style="display:none">
  <div class="flex justify-between" style="margin-bottom:16px">
  <h2>👤 账号管理</h2>
  <div style="display:flex;gap:8px">
    <button class="btn btn-sm" onclick="exportAccounts()">📤 导出账号</button>
    <button class="btn btn-primary btn-sm" onclick="switchTab('import')">➕ 添加</button>
    <button class="btn btn-sm" onclick="loadAccounts()">🔄 刷新</button>
  </div>
</div>
<div class="hint" style="margin:-4px 0 16px;padding:11px 14px;border:1px solid var(--border);border-radius:10px;background:rgba(148,163,184,.05)">
  ℹ️ <strong style="color:var(--text)">Tokens</strong>为本代理本地统计（输入+输出，上游返回 usage 时精确，否则按请求体估算），用于估算离官方限流还有多远；⚡ 测试按钮发起真实探测请求；↻ 重置按钮会<strong style="color:var(--text)">探测上游限流状态</strong>：若上游仍限流则保持冷却并提示恢复时间，探测通过才解除冷却并重置今日统计。
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
<h2>📥 导入账号</h2>
<div class="section">
  <div class="tabs" id="importTabs">
    <div class="tab active" data-tab="oauth">🔑 OAuth 浏览器登录</div>
    <div class="tab" data-tab="token">✏️ 手动输入 Token</div>
    <div class="tab" data-tab="batch">📦 批量导入</div>
  </div>

  <div id="import-oauth" class="tab-content active">
    <p class="hint">通过浏览器完成 OAuth 认证，支持 Google/GitHub/邮箱登录，自动获取 refreshToken。</p>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="startOAuth()" id="oauthBtn">🚀 开始 OAuth 登录</button>
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
      <button class="btn btn-primary" onclick="addByToken()">➕ 添加账号</button>
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
      <button class="btn btn-primary" onclick="batchImport()">📦 导入全部</button>
      <button class="btn" onclick="document.getElementById('fileInput2').click()">📄 选择文件</button>
      <input type="file" id="fileInput2" accept=".json,.txt" style="display:none" onchange="handleFileImport(event)">
    </div>
    <div id="batchResult" style="margin-top:8px"></div>
  </div>
</div>
</div>

<div id="tab-settings" class="tab-panel" style="display:none">
<h2>⚙️ 设置</h2>

<div class="section">
  <div class="section-title">🔑 API 密钥管理</div>
  <div class="section-body">
    <p class="hint">生成的密钥可用于客户端访问代理 API（作为 x-api-key 或 Authorization 头）。</p>
    <div class="form-actions" style="margin-bottom:14px">
      <button class="btn btn-success" onclick="generateKey()">➕ 生成新密钥</button>
    </div>
    <div id="keysList"></div>
    <div id="keyGenResult" style="margin-top:8px"></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🧠 可用模型 <span id="modelsProbeInfo" class="probe-pill" style="font-weight:normal"></span></div>
  <div class="section-body">
    <div class="flex" style="margin-bottom:10px;gap:10px">
      <button class="btn btn-sm btn-primary" onclick="refreshModels()">🔄 刷新模型</button>
      <span class="hint" style="margin:0">自动同步上游官方免费模型（60 秒），仅显示不消耗额度的模型</span>
    </div>
    <div id="modelsList">加载中...</div>
  </div>
</div>

<div class="section">
  <div class="section-title">🔧 代理配置</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>监听地址</label><input type="text" id="settingAddr" disabled></div>
      <div class="field">
        <label>默认模型</label>
        <div style="display:flex;gap:6px;align-items:center">
          <select id="settingDefModel" style="flex:1;font-family:'JetBrains Mono',Consolas,monospace"></select>
          <button class="btn btn-sm btn-primary" onclick="saveDefaultModel()">💾 保存</button>
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
  <div class="section-title">📨 请求头配置（模拟 Cline CLI 发出）</div>
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
      <button class="btn btn-sm" onclick="addHeaderRow()">➕ 添加请求头</button>
      <button class="btn btn-sm btn-primary" onclick="saveHeaders()">💾 保存请求头</button>
    </div>
    <div class="hint">这些请求头会附加到所有转发给 Cline API 的请求中，以模拟官方客户端行为。</div>
    <div id="headerSaveResult" style="margin-top:8px"></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🗑️ 危险操作</div>
  <div class="section-body">
    <div style="display:flex;gap:10px;flex-wrap:wrap">
      <button class="btn btn-danger" onclick="deleteAllAccounts()">🗑️ 删除全部账号</button>
      <button class="btn btn-danger" onclick="deleteAllKeys()">🗑️ 删除全部密钥</button>
    </div>
  </div>
</div>
</div>

<div id="tab-logs" class="tab-panel" style="display:none">
<div class="flex justify-between" style="margin-bottom:16px">
  <h2>📜 请求日志 <span class="probe-pill" style="font-weight:normal">最近 500 条，落盘 data/requests.jsonl</span></h2>
  <div style="display:flex;gap:8px">
    <button class="btn btn-sm" onclick="loadLogs()">🔄 刷新</button>
  </div>
</div>
<div class="section">
  <div class="section-body" style="padding:6px">
    <div class="table-wrap">
    <table>
      <thead>
        <tr><th>时间</th><th>来源</th><th>方法</th><th>路径</th><th>模型</th><th>路由</th><th>Upstream</th><th>Pipeline</th><th>Mode</th><th>Actual Provider</th><th>状态</th><th>耗时</th></tr>
      </thead>
      <tbody id="logsTableBody">
        <tr><td colspan="12" class="empty">加载中...</td></tr>
      </tbody>
    </table>
    </div>
  </div>
</div>
</div>

<div id="tab-opencode" class="tab-panel" style="display:none">
<h2>🌐 opencode 免费模型（统一网关）</h2>

<div class="section">
  <div class="section-title">🔄 上游配置</div>
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
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">💾 保存配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🛡️ 限流防御</div>
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
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">💾 保存限流配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🗜️ 上下文压缩（opencode 官方机制）</div>
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
    <div class="form-actions"><button class="btn btn-primary" onclick="saveOcConfig()">💾 保存压缩配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🧠 opencode 模型列表
    <button class="btn btn-sm" onclick="refreshOcModels()" style="margin-left:auto">🔄 手动同步</button>
  </div>
  <div class="section-body" style="padding:6px">
    <div id="ocModelsList" style="padding:12px">加载中...</div>
  </div>
</div>

<div class="section">
  <div class="section-title">📊 opencode 统计</div>
  <div class="section-body">
    <div class="table-wrap"><div id="ocStatsBox"></div></div>
    <div id="ocModelStatsBox" style="margin-top:14px"></div>
  </div>
</div>
</div>

<div id="tab-clinepass" class="tab-panel" style="display:none">
<h2>🧭 ClinePass Provider Routing</h2>
<div class="hint" style="margin:-8px 0 16px">ClinePass API Key、Provider 策略和 Cline OAuth/Codex 账号完全独立。默认由服务器策略决定 routing，客户端字段不会覆盖服务器配置。</div>

<div class="section">
  <div class="section-title">⚙️ ClinePass 配置</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>启用 ClinePass</label><select id="cpEnabled"><option value="true">开启</option><option value="false">关闭</option></select></div>
      <div class="field"><label>账号模式</label><select id="cpAccountMode"><option value="single">single</option><option value="round_robin">round_robin</option></select></div>
      <div class="field"><label>允许客户端覆盖 Provider</label><select id="cpAllowOverride"><option value="false">关闭（推荐）</option><option value="true">开启</option></select></div>
    </div>
    <div class="form-row"><div class="field"><label>Chat Completions Endpoint</label><input id="cpBaseURL" type="text" placeholder="https://api.cline.bot/api/v1/chat/completions"></div><div class="field"><label>当前单账号</label><span id="cpCurrentAccount" class="hint">未选择</span></div></div>
    <div class="form-actions"><button class="btn btn-primary" onclick="saveClinePassConfig()">💾 保存配置</button></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🔑 API Key 管理</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>名称</label><input id="cpAccountName" placeholder="deepseek-main"></div>
      <div class="field"><label>API Key</label><input id="cpAccountKey" type="password" placeholder="sk-..."></div>
      <div class="field"><label>状态</label><select id="cpAccountEnabled"><option value="true">启用</option><option value="false">停用</option></select></div>
    </div>
    <div class="form-actions"><button class="btn btn-primary" onclick="addClinePassAccount()">➕ 添加账号</button><button class="btn" onclick="loadClinePassAccounts()">🔄 刷新</button></div>
    <div class="table-wrap" style="margin-top:14px"><table><thead><tr><th>#</th><th>名称</th><th>API Key（脱敏）</th><th>状态</th><th>用量</th><th>最后使用</th><th>操作</th></tr></thead><tbody id="cpAccountsBody"><tr><td colspan="7" class="empty">加载中...</td></tr></tbody></table></div>
  </div>
</div>

<div class="section">
  <div class="section-title">🧩 模型 Provider 策略</div>
  <div class="section-body">
    <div class="form-row">
      <div class="field"><label>模型 ID</label><input id="cpModel" placeholder="cline-pass/deepseek-v4.1-flash"></div>
      <div class="field"><label>Pipeline</label><select id="cpPipeline"><option value="auto">auto</option><option value="planner">planner</option><option value="direct">direct</option></select></div>
      <div class="field"><label>Routing Mode</label><select id="cpMode"><option value="auto">auto</option><option value="strict">strict</option><option value="preferred">preferred</option></select></div>
    </div>
    <div class="form-row">
      <div class="field"><label>Providers（逗号分隔，按优先级）</label><input id="cpProviders" placeholder="deepseek,novita,fireworks"></div>
      <div class="field"><label>Excluded（逗号分隔）</label><input id="cpExclude" placeholder="provider-to-skip"></div>
      <div class="field"><label>Sort</label><select id="cpSort"><option value="none">none</option><option value="cost">cost</option><option value="ttft">ttft</option><option value="tps">tps</option></select></div>
    </div>
    <div class="form-actions"><button class="btn btn-primary" onclick="saveClinePassPolicy()">💾 保存策略</button><button class="btn" onclick="probeClinePassModel()">🔎 探测 Provider</button><button class="btn" onclick="validateClinePassModel()">✅ 校验 Provider</button></div>
    <div id="cpProbeResult" class="hint"></div>
    <div class="table-wrap" style="margin-top:14px"><table><thead><tr><th>Model</th><th>Pipeline</th><th>Mode</th><th>Providers</th><th>Excluded</th><th>Actual Provider</th><th>Provider Status</th><th>操作</th></tr></thead><tbody id="cpModelsBody"><tr><td colspan="8" class="empty">加载中...</td></tr></tbody></table></div>
  </div>
</div>
</div>

<div id="tab-codex" class="tab-panel" style="display:none">
<h2>🚀 Codex 上游（伪装 Codex 桌面版客户端）</h2>

<div class="section">
  <div class="section-title">⚙️ Codex 配置</div>
  <div class="section-body">
    <div style="display:grid;gap:12px">
      <label style="display:flex;align-items:center;gap:8px">
        <input type="checkbox" id="codex-enabled" style="width:18px;height:18px">
        <span>启用 Codex 上游</span>
      </label>
      <div>
        <label style="font-size:12px;color:var(--text2)">Originator（客户端标识）</label>
        <input type="text" id="codex-originator" class="input" style="width:100%;margin-top:4px" value="codex_chatgpt_desktop">
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:8px">
        <div>
          <label style="font-size:12px;color:var(--text2)">客户端版本</label>
          <input type="text" id="codex-client-version" class="input" style="width:100%;margin-top:4px" value="0.154.0">
        </div>
        <div>
          <label style="font-size:12px;color:var(--text2)">OS 类型</label>
          <input type="text" id="codex-os-type" class="input" style="width:100%;margin-top:4px" value="Windows">
        </div>
        <div>
          <label style="font-size:12px;color:var(--text2)">OS 版本</label>
          <input type="text" id="codex-os-version" class="input" style="width:100%;margin-top:4px" value="11">
        </div>
      </div>
      <div>
        <label style="font-size:12px;color:var(--text2)">架构</label>
        <input type="text" id="codex-arch" class="input" style="width:100%;margin-top:4px" value="x86_64">
      </div>
      <button onclick="saveCodexConfig()" class="btn" style="margin-top:4px">保存配置</button>
    </div>
  </div>
</div>

<div class="section">
  <div class="section-title">📋 Codex 账号管理</div>
  <div class="section-body">
    <div style="margin-bottom:12px">
      <details>
        <summary style="cursor:pointer;font-size:13px;font-weight:600">导入 auth.json / 手动添加 OAuth 账号</summary>
        <div style="margin-top:12px;display:grid;gap:8px">
          <div>
            <label style="font-size:12px;color:var(--text2)">粘贴 ~/.codex/auth.json 内容（自动解析 refresh_token + account_id）</label>
            <textarea id="codex-auth-json" placeholder='{"tokens":{"refresh_token":"...","access_token":"...","account_id":"..."}}' class="input" style="width:100%;margin-top:4px;height:80px;font-family:monospace;font-size:11px"></textarea>
          </div>
          <button onclick="addCodexAccountFromJSON()" class="btn">从 auth.json 导入</button>
          <hr style="border-color:var(--border);margin:8px 0">
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px">
            <div>
              <label style="font-size:12px;color:var(--text2)">Refresh Token</label>
              <input type="password" id="codex-refresh-token" class="input" style="width:100%;margin-top:4px">
            </div>
            <div>
              <label style="font-size:12px;color:var(--text2)">Account ID（可选，自动从 JWT 提取）</label>
              <input type="text" id="codex-account-id" class="input" style="width:100%;margin-top:4px">
            </div>
          </div>
          <button onclick="addCodexAccountManual()" class="btn">手动添加</button>
        </div>
      </details>
      <details style="margin-top:12px">
        <summary style="cursor:pointer;font-size:13px;font-weight:600">添加自定义供应商（第三方 URL + API Key）</summary>
        <div style="margin-top:12px;display:grid;gap:8px">
          <div>
            <label style="font-size:12px;color:var(--text2)">Base URL</label>
            <input type="text" id="codex-custom-url" placeholder="https://your-provider.com/v1" class="input" style="width:100%;margin-top:4px">
          </div>
          <div>
            <label style="font-size:12px;color:var(--text2)">API Key</label>
            <input type="password" id="codex-custom-apikey" placeholder="sk-..." class="input" style="width:100%;margin-top:4px">
          </div>
          <div>
            <label style="font-size:12px;color:var(--text2)">启用的模型</label>
            <div style="display:flex;gap:8px;align-items:center;margin-top:4px">
              <button type="button" class="btn btn-sm" onclick="fetchCodexModelsForAdd()">获取模型列表</button>
              <span id="codex-custom-models-status" style="font-size:12px;color:var(--text2)"></span>
            </div>
            <div id="codex-custom-models-box" style="margin-top:8px;max-height:200px;overflow-y:auto;border:1px solid var(--border);border-radius:var(--radius-sm);padding:8px;display:none"></div>
          </div>
          <div>
            <label style="font-size:12px;color:var(--text2)">模型映射（JSON，如 {"gpt-5":"gpt-5-turbo"}，可选）</label>
            <textarea id="codex-custom-mapping" placeholder='{"gpt-5":"gpt-5-turbo"}' class="input" style="width:100%;margin-top:4px;height:60px;font-family:monospace;font-size:11px"></textarea>
          </div>
          <button onclick="addCodexCustomAccount()" class="btn">添加供应商</button>
        </div>
      </details>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr><th>#</th><th>类型</th><th>标识</th><th>模型</th><th>状态</th><th>用量</th><th>操作</th></tr></thead>
        <tbody id="codex-accounts-tbody"></tbody>
      </table>
    </div>
  </div>
</div>
</div>
</div>

<div id="codex-edit-modal" style="display:none;position:fixed;inset:0;background:rgba(0,0,0,.5);z-index:1000;align-items:center;justify-content:center">
  <div style="background:var(--bg2);border:1px solid var(--border-strong);border-radius:var(--radius);padding:24px;max-width:560px;width:90%;max-height:80vh;overflow-y:auto">
    <h3 style="margin-bottom:16px;font-size:16px">编辑供应商</h3>
    <div style="display:grid;gap:12px">
      <div>
        <label style="font-size:12px;color:var(--text2)">Base URL</label>
        <input type="text" id="codex-edit-url" class="input" style="width:100%;margin-top:4px">
      </div>
      <div>
        <label style="font-size:12px;color:var(--text2)">API Key（留空不修改）</label>
        <input type="password" id="codex-edit-apikey" class="input" style="width:100%;margin-top:4px" placeholder="留空不修改">
      </div>
      <div>
        <label style="font-size:12px;color:var(--text2)">启用的模型</label>
        <div style="display:flex;gap:8px;align-items:center;margin-top:4px">
          <button type="button" class="btn btn-sm" onclick="fetchCodexModelsForEdit()">获取模型列表</button>
          <span id="codex-edit-models-status" style="font-size:12px;color:var(--text2)"></span>
        </div>
        <div id="codex-edit-models-box" style="margin-top:8px;max-height:200px;overflow-y:auto;border:1px solid var(--border);border-radius:var(--radius-sm);padding:8px"></div>
      </div>
      <div>
        <label style="font-size:12px;color:var(--text2)">模型映射（JSON，可选）</label>
        <textarea id="codex-edit-mapping" class="input" style="width:100%;margin-top:4px;height:60px;font-family:monospace;font-size:11px" placeholder='{"gpt-5":"gpt-5-turbo"}'></textarea>
      </div>
      <div style="display:flex;gap:8px;justify-content:flex-end;margin-top:8px">
        <button class="btn" onclick="closeCodexEditModal()">取消</button>
        <button class="btn btn-primary" onclick="saveCodexCustomEdit()">保存</button>
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
    if (ic) ic.textContent = '🌙';
  } else {
    document.documentElement.setAttribute('data-theme', 'light');
    const ic = document.getElementById('themeIcon');
    if (ic) ic.textContent = '☀️';
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
if (window.location.host) _('footerApiAddr').textContent = 'http://' + window.location.host;

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
    if (el.dataset.tab === 'clinepass') { loadClinePassConfig(); loadClinePassAccounts(); loadClinePassModels(); }
    if (el.dataset.tab === 'codex') { loadCodexConfig(); loadCodexAccounts(); }
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
  if (name === 'clinepass') { loadClinePassConfig(); loadClinePassAccounts(); loadClinePassModels(); }
  if (name === 'codex') { loadCodexConfig(); loadCodexAccounts(); }
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
          '<button class="btn btn-sm" onclick="testAccount(\'' + a.accountId + '\', this)" title="测试账号是否可用（成功会清除冷却/过期状态）">⚡</button> ' +
          '<button class="btn btn-sm" onclick="resetAccount(\'' + a.accountId + '\', this)" title="检测限流并解除：探测上游，若仍限流则保持冷却并提示恢复时间">↻</button> ' +
          '<button class="btn btn-sm btn-danger" onclick="deleteAccount(\'' + a.accountId + '\')" title="删除">✕</button>' +
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
  if (!confirm('⚠️ 确定删除所有账号？不可撤销！')) return;
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
          btn.innerHTML = '🚀 开始 OAuth 登录';
          if (r.data.success) {
            _('oauthProgress').style.display = 'none';
            _('oauthResult').innerHTML = '<div style="color:var(--accent2);font-weight:600;font-size:14px">✓ 账号添加成功: ' + esc(r.data.email) + '</div>';
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
    btn.innerHTML = '🚀 开始 OAuth 登录';
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
      el.innerHTML = '<div class="empty-state"><span class="icon">🔑</span>暂无 API 密钥</div>';
      return;
    }
    el.innerHTML = keys.map(k =>
      '<div class="flex" style="margin-bottom:8px">' +
        '<span class="key-display" style="flex:1" onclick="copyText(\'' + k + '\')" title="点击复制">' + esc(k) + '</span>' +
        '<button class="btn btn-sm btn-danger" onclick="deleteKey(\'' + k + '\')">✕</button>' +
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
        '<div style="color:var(--accent2);font-weight:600;margin-bottom:8px">✓ 新密钥已生成（点击复制）</div>' +
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
const ROUTE_LABEL = { zen: 'opencode', cline: 'cline 池', clinepass: 'ClinePass', admin: '管理', meta: '元信息', other: '其他' };
const STATUS_CLASS = s => s >= 500 ? 'color:var(--danger)' : (s >= 400 ? 'color:var(--amber)' : 'color:var(--accent2)');

async function loadLogs() {
  try {
    const d = await api('GET', '/logs');
    const logs = d.data.logs || [];
    const tbody = _('logsTableBody');
    if (!logs.length) { tbody.innerHTML = '<tr><td colspan="12" class="empty">暂无请求记录</td></tr>'; return; }
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
        '<td>' + esc(l.upstream || '-') + '</td>' +
        '<td>' + esc(l.pipeline || '-') + '</td>' +
        '<td>' + esc(l.routingMode || '-') + '</td>' +
        '<td>' + esc(l.actualProvider || '-') + '</td>' +
        '<td style="font-weight:600;color:' + STATUS_CLASS(st) + '">' + st + '</td>' +
        '<td class="mono" style="font-size:11px">' + (l.duration_ms != null ? l.duration_ms + ' ms' : '-') + '</td>' +
      '</tr>';
    }).join('');
  } catch (e) { tbody.innerHTML = '<tr><td colspan="12" class="empty">加载失败</td></tr>'; }
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
    '<td><button class="btn btn-sm btn-danger" onclick="this.closest(\'tr\').remove()">✕</button></td>';
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
      '<div style="color:var(--accent2);font-size:12px">✓ 已保存 ' + Object.keys(d.data.headers).length + ' 个请求头</div>';
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
    toast(d.data.message || '同步已开始', 'info');
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
          '<td><button class="btn btn-sm btn-danger" onclick="this.closest(\'tr\').remove()">✕</button></td>' +
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
      ? '<span style="color:var(--danger)">🔴 故障转移中 (opencode 不可用, 请求走 cline 池)</span>'
      : '<span style="color:var(--accent2)">🟢 正常</span>';
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

// ========== ClinePass Provider 管理 ==========
async function loadClinePassConfig() {
  try {
    const d = await api('GET', '/clinepass/config');
    const c = d.data || {};
    _('cpEnabled').value = String(c.enabled);
    _('cpAccountMode').value = c.accountMode || 'round_robin';
    _('cpAllowOverride').value = String(c.allowClientProviderOverride);
    _('cpBaseURL').value = c.baseURL || '';
  } catch (e) { /* ignore */ }
}
async function saveClinePassConfig() {
  try {
    await api('POST', '/clinepass/config/update', {
      enabled: _('cpEnabled').value === 'true',
      accountMode: _('cpAccountMode').value,
      allowClientProviderOverride: _('cpAllowOverride').value === 'true',
      baseURL: _('cpBaseURL').value.trim()
    });
    toast('ClinePass 配置已保存', 'success');
  } catch (e) { toast('保存失败: ' + e.message, 'error'); }
}
async function loadClinePassAccounts() {
  try {
    const d = await api('GET', '/clinepass/accounts');
    const list = d.data || [];
    const tb = _('cpAccountsBody');
    const current = list.find(a => a.current);
    if (_('cpCurrentAccount')) _('cpCurrentAccount').textContent = current ? current.name : '未选择';
    if (!list.length) { tb.innerHTML = '<tr><td colspan="7" class="empty">暂无 ClinePass API Key</td></tr>'; return; }
    tb.innerHTML = list.map(a => '<tr>' +
      '<td>' + a.index + '</td><td>' + esc(a.name) + '</td><td class="mono">' + esc(a.apiKey) + '</td>' +
      '<td><span class="badge ' + (a.enabled ? 'badge-ok' : 'badge-err') + '">' + (a.enabled ? '启用' : '停用') + '</span></td>' +
      '<td>' + (a.usageCount || 0) + '</td><td>' + (a.lastUsed ? new Date(a.lastUsed).toLocaleString('zh-CN') : '-') + '</td>' +
      '<td>' + (a.current ? '<span class="badge badge-ok">当前</span> ' : '') + '<button class="btn-sm" onclick="setClinePassCurrentAccount(' + a.index + ')">设为当前</button> <button class="btn-sm" onclick="testClinePassAccount(' + a.index + ')">测试</button> <button class="btn-sm" onclick="toggleClinePassAccount(' + a.index + ',' + (!a.enabled) + ')">' + (a.enabled ? '停用' : '启用') + '</button> <button class="btn-sm btn-danger" onclick="deleteClinePassAccount(' + a.index + ')">删除</button></td>' +
      '</tr>').join('');
  } catch (e) { _('cpAccountsBody').innerHTML = '<tr><td colspan="7" class="empty">加载失败</td></tr>'; }
}
async function setClinePassCurrentAccount(index) {
  try { await api('POST', '/clinepass/config/update', {currentIdx:index}); toast('当前单账号已更新', 'success'); loadClinePassConfig(); loadClinePassAccounts(); }
  catch (e) { toast('设置失败: ' + e.message, 'error'); }
}
async function addClinePassAccount() {
  const name = _('cpAccountName').value.trim(), apiKey = _('cpAccountKey').value.trim();
  if (!name || !apiKey) { toast('名称和 API Key 不能为空', 'error'); return; }
  try {
    await api('POST', '/clinepass/accounts/add', {name, apiKey, enabled: _('cpAccountEnabled').value === 'true'});
    _('cpAccountName').value = ''; _('cpAccountKey').value = '';
    toast('ClinePass 账号已添加', 'success'); loadClinePassAccounts();
  } catch (e) { toast('添加失败: ' + e.message, 'error'); }
}
async function toggleClinePassAccount(index, enabled) {
  try { await api('POST', '/clinepass/accounts/update', {index, enabled}); loadClinePassAccounts(); }
  catch (e) { toast('更新失败: ' + e.message, 'error'); }
}
async function deleteClinePassAccount(index) {
  if (!confirm('确认删除此 ClinePass API Key？')) return;
  try { await api('POST', '/clinepass/accounts/delete', {index}); toast('账号已删除', 'success'); loadClinePassAccounts(); }
  catch (e) { toast('删除失败: ' + e.message, 'error'); }
}
async function testClinePassAccount(index) {
  try { const d = await api('POST', '/clinepass/accounts/test', {index, model: _('cpModel').value.trim()}); toast('测试结果: ' + (d.data.status || 'ok'), d.data.status === 'ok' ? 'success' : 'warning'); }
  catch (e) { toast('测试失败: ' + e.message, 'error'); }
}
function splitProviders(value) { return value.split(',').map(s => s.trim()).filter(Boolean); }
function setClinePassPolicyForm(model, policy) {
  _('cpModel').value = model || '';
  _('cpPipeline').value = policy.pipeline || 'auto';
  _('cpMode').value = policy.mode || 'auto';
  _('cpProviders').value = (policy.providers || []).join(',');
  _('cpExclude').value = (policy.exclude || []).join(',');
  _('cpSort').value = policy.sort || 'none';
}
let clinePassModelCache = {};
function selectClinePassModel(model) {
  const item = clinePassModelCache[model];
  if (item) setClinePassPolicyForm(model, {pipeline:item.pipeline, mode:item.mode, providers:item.providers||[], exclude:item.exclude||[], sort:item.sort||'none'});
}
async function loadClinePassModels() {
  try {
    const d = await api('GET', '/clinepass/models');
    const list = (d.data && d.data.models) || [];
    const tb = _('cpModelsBody');
    if (!list.length) { tb.innerHTML = '<tr><td colspan="8" class="empty">暂无模型策略。可先填写模型 ID 并保存。</td></tr>'; return; }
    clinePassModelCache = {};
    list.forEach(m => { clinePassModelCache[m.id] = m; });
    tb.innerHTML = list.map(m => {
      const statuses = m.providerStatus || {};
      const statusText = Object.keys(statuses).map(k => esc(k) + ':' + esc(statuses[k])).join(', ') || '-';
      const modelArg = String(m.id).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
      return '<tr>' +
        '<td class="mono">' + esc(m.id) + '</td><td>' + esc(m.pipeline || 'unknown') + '</td><td>' + esc(m.mode || 'auto') + '</td>' +
        '<td>' + (m.providers || []).map(p => '<span class="model-tag pass">' + esc(p) + '</span>').join('') + '</td>' +
        '<td>' + (m.exclude || []).map(p => '<span class="model-tag">' + esc(p) + '</span>').join('') + '</td>' +
        '<td>' + esc(m.actualProvider || '-') + '</td><td class="mono">' + statusText + '</td><td><button class="btn-sm" onclick="selectClinePassModel(\'' + modelArg + '\')">编辑</button></td></tr>';
    }).join('');
  } catch (e) { _('cpModelsBody').innerHTML = '<tr><td colspan="8" class="empty">加载失败</td></tr>'; }
}
async function saveClinePassPolicy() {
  const model = _('cpModel').value.trim();
  if (!model) { toast('请输入模型 ID', 'error'); return; }
  const policy = {pipeline:_('cpPipeline').value, mode:_('cpMode').value, providers:splitProviders(_('cpProviders').value), exclude:splitProviders(_('cpExclude').value), sort:_('cpSort').value};
  try { await api('POST', '/clinepass/models/policy', {model, policy}); toast('模型策略已保存', 'success'); loadClinePassModels(); }
  catch (e) { toast('保存策略失败: ' + e.message, 'error'); }
}
async function probeClinePassModel() {
  const model = _('cpModel').value.trim();
  if (!model) { toast('请输入模型 ID', 'error'); return; }
  _('cpProbeResult').textContent = '探测中...';
  try {
    const d = await api('POST', '/clinepass/models/probe', {model, pipeline:_('cpPipeline').value});
    const x = d.data || {}; _('cpProviders').value = (x.providers || []).join(',');
    _('cpPipeline').value = x.pipeline || _('cpPipeline').value;
    _('cpProbeResult').textContent = '发现 Provider: ' + ((x.providers || []).join(', ') || 'unknown') + ' · Pipeline: ' + (x.pipeline || 'unknown');
    loadClinePassModels();
  } catch (e) { _('cpProbeResult').textContent = '探测失败: ' + e.message; }
}
async function validateClinePassModel() {
  const model = _('cpModel').value.trim();
  if (!model) { toast('请输入模型 ID', 'error'); return; }
  try {
    const d = await api('POST', '/clinepass/models/validate', {model, providers:splitProviders(_('cpProviders').value)});
    const statuses = d.data.statuses || {}; _('cpProbeResult').textContent = Object.keys(statuses).map(k => k + ': ' + statuses[k]).join(' · '); loadClinePassModels();
  } catch (e) { toast('校验失败: ' + e.message, 'error'); }
}

// ========== Codex 上游管理 ==========
function loadCodexConfig() {
  fetch(API + '/codex/config').then(r => r.json()).then(r => {
    if (!r.success) return;
    const d = r.data;
    _('codex-enabled').checked = d.enabled;
    _('codex-originator').value = d.originator || 'codex_chatgpt_desktop';
    _('codex-client-version').value = d.clientVersion || '0.154.0';
    _('codex-os-type').value = d.osType || 'Windows';
    _('codex-os-version').value = d.osVersion || '11';
    _('codex-arch').value = d.arch || 'x86_64';
  });
}
function saveCodexConfig() {
  const body = {
    enabled: _('codex-enabled').checked,
    originator: _('codex-originator').value,
    clientVersion: _('codex-client-version').value,
    osType: _('codex-os-type').value,
    osVersion: _('codex-os-version').value,
    arch: _('codex-arch').value,
  };
  fetch(API + '/codex/config/update', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)}).then(r=>r.json()).then(r=>{
    toast(r.success ? 'Codex 配置已保存' : (r.error||'保存失败'));
    if (r.success) loadCodexConfig();
  });
}
function loadCodexAccounts() {
  fetch(API + '/codex/accounts').then(r=>r.json()).then(r=>{
    if (!r.success) return;
    const tb = _('codex-accounts-tbody');
    tb.innerHTML = '';
    (r.data||[]).forEach(a => {
      const isCustom = a.type === 'custom';
      const ident = isCustom ? esc((a.customUrl||'').slice(0,40)) : esc(a.email||'-');
      const models = isCustom ? (a.models||[]).join(', ') || '(全部)' : '-';
      const actions = isCustom
        ? '<button class="btn-sm" onclick="editCodexCustomAccount('+a.index+')">编辑</button> <button class="btn-sm btn-danger" onclick="deleteCodexAccount('+a.index+')">删除</button>'
        : '<button class="btn-sm" onclick="refreshCodexAccount('+a.index+')">刷新</button> <button class="btn-sm btn-danger" onclick="deleteCodexAccount('+a.index+')">删除</button>';
      const tr = document.createElement('tr');
      tr.innerHTML = '<td>'+a.index+'</td><td><span class="badge '+(isCustom?'badge-err':'badge-ok')+'">'+esc(a.type)+'</span></td><td style="font-family:monospace;font-size:11px">'+ident+'</td><td style="font-size:11px">'+esc(models)+'</td><td><span class="badge '+(a.status==='active'?'badge-ok':'badge-err')+'">'+esc(a.status)+'</span></td><td>'+a.usageCount+'</td><td>'+actions+'</td>';
      tb.appendChild(tr);
    });
  });
}
function addCodexAccountFromJSON() {
  const raw = _('codex-auth-json').value.trim();
  if (!raw) { toast('请粘贴 auth.json 内容'); return; }
  fetch(API + '/codex/accounts/add', {method:'POST', headers:{'Content-Type':'application/json'}, body:raw}).then(r=>r.json()).then(r=>{
    toast(r.success ? '账号导入成功' : (r.error||'导入失败'));
    if (r.success) { _('codex-auth-json').value=''; loadCodexAccounts(); }
  });
}
function addCodexAccountManual() {
  const body = {refreshToken: _('codex-refresh-token').value, accountId: _('codex-account-id').value};
  if (!body.refreshToken) { toast('请输入 Refresh Token'); return; }
  fetch(API + '/codex/accounts/add', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)}).then(r=>r.json()).then(r=>{
    toast(r.success ? '账号添加成功' : (r.error||'添加失败'));
    if (r.success) { _('codex-refresh-token').value=''; _('codex-account-id').value=''; loadCodexAccounts(); }
  });
}
// 渲染模型勾选列表
function renderCodexModelCheckboxes(boxId, models, selected) {
  const box = _(boxId);
  if (!models || models.length === 0) {
    box.innerHTML = '<span style="font-size:12px;color:var(--text2)">无可用模型</span>';
    box.style.display = 'block';
    return;
  }
  const selSet = new Set(selected || []);
  box.innerHTML = models.map(m => {
    const checked = selSet.has(m) ? 'checked' : '';
    return '<label style="display:flex;align-items:center;gap:6px;padding:4px 0;font-size:13px;cursor:pointer"><input type="checkbox" value="'+esc(m)+'" '+checked+' style="width:16px;height:16px"> '+esc(m)+'</label>';
  }).join('');
  box.style.display = 'block';
}
// 获取勾选的模型列表
function getCheckedModels(boxId) {
  const box = _(boxId);
  return Array.from(box.querySelectorAll('input[type=checkbox]:checked')).map(cb => cb.value);
}
// 添加供应商: 获取模型列表
function fetchCodexModelsForAdd() {
  const url = _('codex-custom-url').value.trim();
  const key = _('codex-custom-apikey').value.trim();
  if (!url) { toast('请输入 Base URL'); return; }
  if (!key) { toast('请输入 API Key'); return; }
  const status = _('codex-custom-models-status');
  status.textContent = '获取中...';
  fetch(API + '/codex/models/fetch', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({url:url, apiKey:key})}).then(r=>r.json()).then(r=>{
    if (r.success) {
      const models = r.data.models || [];
      status.textContent = '获取到 ' + models.length + ' 个模型';
      renderCodexModelCheckboxes('codex-custom-models-box', models, models);
    } else {
      status.textContent = '';
      toast(r.error || '获取失败');
    }
  }).catch(e => { status.textContent = ''; toast('网络错误'); });
}
function addCodexCustomAccount() {
  const url = _('codex-custom-url').value.trim();
  const key = _('codex-custom-apikey').value.trim();
  const mappingStr = _('codex-custom-mapping').value.trim();
  if (!url) { toast('请输入 Base URL'); return; }
  if (!key) { toast('请输入 API Key'); return; }
  const models = getCheckedModels('codex-custom-models-box');
  let modelMapping = {};
  if (mappingStr) {
    try { modelMapping = JSON.parse(mappingStr); } catch(e) { toast('模型映射 JSON 格式错误'); return; }
  }
  const body = {type:'custom', customUrl:url, customApiKey:key, models:models, modelMapping:modelMapping};
  fetch(API + '/codex/accounts/add', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)}).then(r=>r.json()).then(r=>{
    toast(r.success ? '供应商添加成功' : (r.error||'添加失败'));
    if (r.success) {
      _('codex-custom-url').value=''; _('codex-custom-apikey').value=''; _('codex-custom-mapping').value='';
      _('codex-custom-models-box').innerHTML=''; _('codex-custom-models-box').style.display='none';
      _('codex-custom-models-status').textContent='';
      loadCodexAccounts();
    }
  });
}
// 编辑供应商: 弹窗
let codexEditIndex = -1;
function editCodexCustomAccount(idx) {
  fetch(API + '/codex/accounts').then(r=>r.json()).then(r=>{
    if (!r.success) return;
    const acc = (r.data||[]).find(a => a.index === idx);
    if (!acc) return;
    codexEditIndex = idx;
    _('codex-edit-url').value = acc.customUrl || '';
    _('codex-edit-apikey').value = '';
    _('codex-edit-mapping').value = acc.modelMapping ? JSON.stringify(acc.modelMapping, null, 2) : '';
    _('codex-edit-models-status').textContent = '';
    // 渲染已启用的模型 (已勾选)
    renderCodexModelCheckboxes('codex-edit-models-box', acc.models || [], acc.models || []);
    _('codex-edit-modal').style.display = 'flex';
  });
}
function closeCodexEditModal() {
  _('codex-edit-modal').style.display = 'none';
  codexEditIndex = -1;
}
function fetchCodexModelsForEdit() {
  const url = _('codex-edit-url').value.trim();
  const key = _('codex-edit-apikey').value.trim();
  if (!url) { toast('请输入 Base URL'); return; }
  const status = _('codex-edit-models-status');
  const prevSelected = getCheckedModels('codex-edit-models-box');
  status.textContent = '获取中...';
  // key 为空时传 index, 后端用存储的 key
  const payload = {url:url};
  if (key) { payload.apiKey = key; } else { payload.index = codexEditIndex; }
  fetch(API + '/codex/models/fetch', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(payload)}).then(r=>r.json()).then(r=>{
    if (r.success) {
      const models = r.data.models || [];
      status.textContent = '获取到 ' + models.length + ' 个模型';
      renderCodexModelCheckboxes('codex-edit-models-box', models, prevSelected);
    } else {
      status.textContent = '';
      toast(r.error || '获取失败');
    }
  }).catch(e => { status.textContent = ''; toast('网络错误'); });
}
function saveCodexCustomEdit() {
  if (codexEditIndex < 0) return;
  const url = _('codex-edit-url').value.trim();
  const key = _('codex-edit-apikey').value.trim();
  const mappingStr = _('codex-edit-mapping').value.trim();
  if (!url) { toast('Base URL 不能为空'); return; }
  const models = getCheckedModels('codex-edit-models-box');
  let modelMapping = {};
  if (mappingStr.trim()) {
    try { modelMapping = JSON.parse(mappingStr); } catch(e) { toast('模型映射 JSON 格式错误'); return; }
  }
  const body = {index:codexEditIndex, customUrl:url, models:models, modelMapping:modelMapping};
  if (key) body.customApiKey = key;
  fetch(API + '/codex/accounts/update', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)}).then(r=>r.json()).then(r=>{
    toast(r.success ? '供应商已更新' : (r.error||'更新失败'));
    if (r.success) { closeCodexEditModal(); loadCodexAccounts(); }
  });
}
function deleteCodexAccount(idx) {
  if (!confirm('确认删除此账号？')) return;
  fetch(API + '/codex/accounts/delete', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({index:idx})}).then(r=>r.json()).then(r=>{
    toast(r.success ? '已删除' : (r.error||'删除失败'));
    if (r.success) loadCodexAccounts();
  });
}
function refreshCodexAccount(idx) {
  fetch(API + '/codex/accounts/refresh', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({index:idx})}).then(r=>r.json()).then(r=>{
    toast(r.success ? 'Token 已刷新' : (r.error||'刷新失败'));
    if (r.success) loadCodexAccounts();
  });
}

// ========== 初始化 ==========
loadStats();
loadAccounts();
loadKeys();
loadModels();
loadConfig();
loadClinePassConfig();
loadClinePassAccounts();
loadClinePassModels();
loadCodexConfig();
loadCodexAccounts();
setInterval(() => { loadStats(); }, 10000);
setInterval(() => { loadOcStats(); }, 15000);
setInterval(() => { if (_('tab-logs').style.display !== 'none') loadLogs(); }, 8000);
setInterval(() => { if (_('tab-clinepass').style.display !== 'none') loadClinePassModels(); }, 15000);
setInterval(() => { if (_('tab-codex').style.display !== 'none') loadCodexAccounts(); }, 15000);
</script>
</body>
</html>`
