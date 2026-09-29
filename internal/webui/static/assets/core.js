/* ServerAdmin 前端核心：状态管理 / API 封装 / 通用组件 / 路由 / 启动流程 */
'use strict';

// ============================ 全局状态 ============================
const state = {
  token: localStorage.getItem('sa_access') || '',
  refresh: localStorage.getItem('sa_refresh') || '',
  user: null,
  roles: [],
  perms: [],
  menus: [],
  routeMap: {},
  publicInfo: { name: 'ServerAdmin', version: '', copyright: '' },
};

// ============================ 工具函数 ============================
function $(sel, root) { return (root || document).querySelector(sel); }
function $$(sel, root) { return Array.from((root || document).querySelectorAll(sel)); }

function esc(s) {
  if (s === null || s === undefined) return '';
  return String(s).replace(/[&<>"']/g, m => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m]));
}

function fmtTime(t) {
  if (!t) return '-';
  const d = new Date(t);
  if (isNaN(d.getTime())) return '-';
  const p = n => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

function fmtSize(bytes) {
  if (bytes === 0) return '0 B';
  if (!bytes) return '-';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
}

function fmtUptime(seconds) {
  if (seconds == null) return '-';
  const d = Math.floor(seconds / 86400), h = Math.floor(seconds % 86400 / 3600), m = Math.floor(seconds % 3600 / 60);
  if (d > 0) return `${d}天${h}小时${m}分`;
  if (h > 0) return `${h}小时${m}分`;
  return `${m}分钟`;
}

function debounce(fn, ms) {
  let t;
  return function (...args) { clearTimeout(t); t = setTimeout(() => fn.apply(this, args), ms); };
}

function hasPerm(perm) {
  if (!perm) return true;
  return state.perms.includes('*') || state.perms.includes(perm);
}

function statusTag(v, onText = '启用', offText = '停用') {
  return v === 1 ? `<span class="tag tag-success">${onText}</span>` : `<span class="tag tag-danger">${offText}</span>`;
}

function tag(text, type) {
  return `<span class="tag tag-${type || 'info'}">${esc(text)}</span>`;
}

// 字典缓存
const dictCache = {};
async function dict(code) {
  if (dictCache[code]) return dictCache[code];
  try {
    const items = await api('/dicts/' + code);
    dictCache[code] = items || [];
  } catch (e) { dictCache[code] = []; }
  return dictCache[code];
}

// ============================ API 封装 ============================
let refreshing = null;

async function tryRefresh() {
  if (!state.refresh) return false;
  if (refreshing) return refreshing;
  refreshing = (async () => {
    try {
      const res = await fetch('/api/v1/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken: state.refresh }),
      });
      const data = await res.json();
      if (data.code !== 0) return false;
      saveTokens(data.data.accessToken, data.data.refreshToken);
      return true;
    } catch (e) { return false; }
    finally { setTimeout(() => { refreshing = null; }, 0); }
  })();
  return refreshing;
}

function saveTokens(access, refresh) {
  state.token = access;
  state.refresh = refresh;
  localStorage.setItem('sa_access', access);
  localStorage.setItem('sa_refresh', refresh);
}

function clearTokens() {
  state.token = '';
  state.refresh = '';
  localStorage.removeItem('sa_access');
  localStorage.removeItem('sa_refresh');
}

async function api(path, { method = 'GET', body, query, retry = true } = {}) {
  let url = '/api/v1' + path;
  if (query) {
    const qs = new URLSearchParams();
    Object.entries(query).forEach(([k, v]) => { if (v !== '' && v !== null && v !== undefined) qs.set(k, v); });
    const s = qs.toString();
    if (s) url += '?' + s;
  }
  const headers = {};
  if (state.token) headers['Authorization'] = 'Bearer ' + state.token;
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  const res = await fetch(url, { method, headers, body: body !== undefined ? JSON.stringify(body) : undefined });

  if (res.status === 401 && retry) {
    const ok = await tryRefresh();
    if (ok) return api(path, { method, body, query, retry: false });
    clearTokens();
    const err = new Error('登录已过期');
    err.silent = true;
    throw err;
  }

  let data;
  try { data = await res.json(); } catch (e) { data = { code: res.status, msg: '服务响应异常' }; }
  if (data.code !== 0) {
    const err = new Error(data.msg || ('请求失败（' + data.code + '）'));
    err.code = data.code;
    throw err;
  }
  return data.data;
}

// ============================ Toast / 弹窗 ============================
function toast(msg, type = 'success') {
  const root = $('#toast-root');
  const el = document.createElement('div');
  el.className = 'toast toast-' + type;
  el.textContent = msg;
  root.appendChild(el);
  setTimeout(() => { el.style.opacity = '0'; el.style.transition = 'opacity .3s'; }, 2400);
  setTimeout(() => el.remove(), 2800);
}
function toastError(e) { if (e && !e.silent) toast(e.message || '操作失败', 'error'); }

function openModal({ title, body, footer, width = '', onClose }) {
  const root = $('#modal-root');
  const mask = document.createElement('div');
  mask.className = 'modal-mask';
  mask.innerHTML = `
    <div class="modal ${width}">
      <div class="modal-head">
        <div class="modal-title">${esc(title)}</div>
        <button class="modal-close" title="关闭">×</button>
      </div>
      <div class="modal-body"></div>
      ${footer ? '<div class="modal-foot"></div>' : ''}
    </div>`;
  $('.modal-body', mask).appendChild(typeof body === 'string' ? htmlToEl(body) : body);
  if (footer) $('.modal-foot', mask).appendChild(typeof footer === 'string' ? htmlToEl(footer) : footer);
  const close = () => { mask.remove(); if (onClose) onClose(); };
  $('.modal-close', mask).onclick = close;
  mask.addEventListener('mousedown', e => { if (e.target === mask) close(); });
  root.appendChild(mask);
  mask.close = close;
  return mask;
}

function htmlToEl(html) {
  const t = document.createElement('template');
  t.innerHTML = html.trim();
  return t.content.firstElementChild;
}

function confirmBox(message, { danger = true, title = '确认操作' } = {}) {
  return new Promise(resolve => {
    const foot = htmlToEl(`
      <div style="display:flex;gap:10px">
        <button class="btn" data-act="cancel">取消</button>
        <button class="btn ${danger ? 'btn-danger' : 'btn-primary'}" data-act="ok">确定</button>
      </div>`);
    const m = openModal({
      title,
      width: 'modal-sm',
      body: `<div style="padding:4px 2px;font-size:14px;line-height:1.7">${message}</div>`,
      footer: foot,
      onClose: () => resolve(false),
    });
    $('[data-act="cancel"]', foot).onclick = () => { m.onClose = null; m.close(); resolve(false); };
    $('[data-act="ok"]', foot).onclick = () => { m.onClose = null; m.close(); resolve(true); };
  });
}

// ============================ 表单弹窗 ============================
// fields: {key,label,type,required,options,value,placeholder,help}
// type: text password number textarea select radio switch checkboxes date
function formModal({ title, fields, values = {}, submitText = '保 存', width = '', onSubmit }) {
  const form = htmlToEl('<form class="modal-form" autocomplete="off"></form>');
  fields.forEach(f => {
    const val = values[f.key] !== undefined && values[f.key] !== null ? values[f.key] : (f.value !== undefined ? f.value : '');
    const item = htmlToEl(`<div class="form-item" data-key="${f.key}"></div>`);
    let ctrl = '';
    const req = f.required ? '<span class="req">*</span>' : '';
    switch (f.type) {
      case 'select':
        ctrl = `<select name="${f.key}">` +
          (f.options || []).map(o => `<option value="${esc(o.value)}" ${String(val) === String(o.value) ? 'selected' : ''}>${esc(o.label)}</option>`).join('') +
          `</select>`;
        break;
      case 'radio':
        ctrl = `<div class="form-radio-group">` + (f.options || []).map(o =>
          `<label class="form-check-item"><input type="radio" name="${f.key}" value="${esc(o.value)}" ${String(val) === String(o.value) ? 'checked' : ''}>${esc(o.label)}</label>`).join('') + `</div>`;
        break;
      case 'switch':
        ctrl = `<label class="form-check-item"><input type="checkbox" name="${f.key}" ${val ? 'checked' : ''}> ${esc(f.switchLabel || '是')}</label>`;
        break;
      case 'checkboxes':
        ctrl = `<div class="form-check-group">` + (f.options || []).map(o =>
          `<label class="form-check-item"><input type="checkbox" name="${f.key}" value="${esc(o.value)}" ${(Array.isArray(val) ? val : []).map(String).includes(String(o.value)) ? 'checked' : ''}>${esc(o.label)}</label>`).join('') + `</div>`;
        break;
      case 'textarea':
        ctrl = `<textarea name="${f.key}" rows="${f.rows || 4}" placeholder="${esc(f.placeholder || '')}">${esc(val)}</textarea>`;
        break;
      default:
        ctrl = `<input type="${f.type || 'text'}" name="${f.key}" value="${esc(val)}" placeholder="${esc(f.placeholder || '')}" ${f.disabled ? 'disabled' : ''}>`;
    }
    item.innerHTML = `<label>${esc(f.label)}${req}</label>${ctrl}${f.help ? `<div class="form-help">${esc(f.help)}</div>` : ''}`;
    form.appendChild(item);
  });

  const foot = htmlToEl(`
    <div style="display:flex;gap:10px">
      <button type="button" class="btn" data-act="cancel">取消</button>
      <button type="submit" class="btn btn-primary">${esc(submitText)}</button>
    </div>`);
  const m = openModal({ title, body: form, footer: foot, width });

  $('[data-act="cancel"]', foot).onclick = () => m.close();
  form.addEventListener('submit', async e => {
    e.preventDefault();
    const values2 = {};
    for (const f of fields) {
      const els = form.querySelectorAll(`[name="${f.key}"]`);
      if (!els.length) { values2[f.key] = f.value !== undefined ? f.value : ''; continue; }
      if (f.type === 'checkboxes') {
        values2[f.key] = Array.from(els).filter(i => i.checked).map(i => i.value);
      } else if (f.type === 'switch') {
        values2[f.key] = els[0].checked;
      } else if (f.type === 'radio') {
        const checked = form.querySelector(`[name="${f.key}"]:checked`);
        values2[f.key] = checked ? checked.value : '';
      } else {
        values2[f.key] = els[0].value.trim();
      }
      // select/radio 的纯数字选项自动转为数值（状态、类型等后端为 int 字段）
      if ((f.type === 'select' || f.type === 'radio') && typeof values2[f.key] === 'string'
        && values2[f.key] !== '' && /^-?\d+$/.test(values2[f.key])) {
        values2[f.key] = Number(values2[f.key]);
      }
      if (f.type === 'number' && values2[f.key] !== '') values2[f.key] = Number(values2[f.key]);
      if (f.required) {
        const v = values2[f.key];
        if (v === '' || v === null || (Array.isArray(v) && !v.length)) {
          toast(`${f.label}不能为空`, 'error');
          return;
        }
      }
    }
    const btn = $('[type="submit"]', foot);
    btn.disabled = true;
    try {
      await onSubmit(values2);
      m.close();
    } catch (err) {
      toastError(err);
    } finally {
      btn.disabled = false;
    }
  });
  return m;
}

// ============================ 表格 / 分页 ============================
function renderTable(cols, rows, { rowKey = 'id', checkbox = false, selected = [] } = {}) {
  const head = (checkbox ? '<th style="width:36px"><input type="checkbox" class="sel-all"></th>' : '') +
    cols.map(c => `<th ${c.width ? `style="width:${c.width}"` : ''}>${c.title}</th>`).join('');
  const body = rows.length
    ? rows.map(r => {
        const cells = cols.map(c => `<td>${c.render ? c.render(r) : esc(r[c.key] !== undefined && r[c.key] !== null ? r[c.key] : '-')}</td>`).join('');
        const sel = checkbox ? `<td><input type="checkbox" class="sel-row" value="${esc(r[rowKey])}" ${selected.includes(r[rowKey]) ? 'checked' : ''}></td>` : '';
        return `<tr data-id="${esc(r[rowKey])}">${sel}${cells}</tr>`;
      }).join('')
    : `<tr><td colspan="${cols.length + (checkbox ? 1 : 0)}" class="empty">暂无数据</td></tr>`;
  return `<div class="table-wrap"><table class="table"><thead><tr>${head}</tr></thead><tbody>${body}</tbody></table></div>`;
}

function renderPager(total, page, size) {
  const pages = Math.max(Math.ceil(total / size), 1);
  return `<div class="pager">
    <span>共 <b>${total}</b> 条</span>
    <button data-p="1" ${page <= 1 ? 'disabled' : ''}>首页</button>
    <button data-p="${page - 1}" ${page <= 1 ? 'disabled' : ''}>上一页</button>
    <span class="current">${page} / ${pages}</span>
    <button data-p="${page + 1}" ${page >= pages ? 'disabled' : ''}>下一页</button>
    <button data-p="${pages}" ${page >= pages ? 'disabled' : ''}>末页</button>
  </div>`;
}

// ============================ 通用列表页 ============================
// cfg: {title, desc, filters, columns, loadData({page,size,...query}), rowActions(row),
//       canAdd, onAdd, checkbox, onBatchDelete(ids), defaultQuery, pageSize}
function listPage(container, cfg) {
  let page = 1;
  let size = cfg.pageSize || 20;
  const query = Object.assign({}, cfg.defaultQuery || {});
  const selected = new Set();

  const view = htmlToEl(`
    <div>
      <div class="page-header">
        <div>
          <div class="page-title">${esc(cfg.title)}</div>
          ${cfg.desc ? `<div class="page-desc">${esc(cfg.desc)}</div>` : ''}
        </div>
        <div class="header-actions"></div>
      </div>
      <div class="card">
        <div class="toolbar"></div>
        <div class="data-area"></div>
      </div>
    </div>`);
  container.appendChild(view);
  const toolbar = $('.toolbar', view);
  const dataArea = $('.data-area', view);
  const headerActions = $('.header-actions', view);

  // 筛选条件
  (cfg.filters || []).forEach(f => {
    if (f.type === 'select') {
      const sel = htmlToEl(`<select data-filter="${f.key}"><option value="">${esc(f.label)}</option>${(f.options || []).map(o => `<option value="${esc(o.value)}">${esc(o.label)}</option>`).join('')}</select>`);
      toolbar.appendChild(sel);
      sel.onchange = () => { query[f.key] = sel.value; page = 1; load(); };
    } else if (f.type === 'date') {
      const inp = htmlToEl(`<input type="date" data-filter="${f.key}" title="${esc(f.label)}">`);
      toolbar.appendChild(inp);
      inp.onchange = () => { query[f.key] = inp.value; page = 1; load(); };
    } else {
      const inp = htmlToEl(`<input type="text" data-filter="${f.key}" placeholder="${esc(f.placeholder || f.label)}" style="width:160px">`);
      toolbar.appendChild(inp);
      inp.oninput = debounce(() => { query[f.key] = inp.value.trim(); page = 1; load(); }, 400);
    }
  });
  if ((cfg.filters || []).length) {
    const reset = htmlToEl('<button class="btn">重置</button>');
    toolbar.appendChild(reset);
    reset.onclick = () => {
      Object.keys(query).forEach(k => delete query[k]);
      $$('[data-filter]', toolbar).forEach(i => { i.value = ''; });
      page = 1; load();
    };
  }
  toolbar.appendChild(htmlToEl('<span class="spacer"></span>'));

  if (cfg.canAdd) {
    const addBtn = htmlToEl(`<button class="btn btn-primary">＋ ${esc(cfg.addText || '新增')}</button>`);
    headerActions.appendChild(addBtn);
    addBtn.onclick = () => cfg.onAdd(load);
  }
  if (cfg.extraActions) cfg.extraActions(headerActions, () => load());

  async function load() {
    dataArea.innerHTML = '<div class="empty" style="padding:30px;color:#94a3b8">加载中...</div>';
    try {
      const res = await cfg.loadData({ page, size, ...query });
      const list = res.list || [];
      const total = res.total || 0;
      dataArea.innerHTML = renderTable(cfg.columns, list, {
        checkbox: !!cfg.checkbox, rowKey: cfg.rowKey || 'id', selected: Array.from(selected),
      }) + renderPager(total, page, size);

      // 操作列（事件委托）
      const actionsHtml = (row) => (cfg.rowActions ? cfg.rowActions(row) : []).map((a, i) =>
        `<button class="btn-link ${a.danger ? 'danger' : ''}" data-act="${i}" data-id="${esc(row[cfg.rowKey || 'id'])}" ${a.perm && !hasPerm(a.perm) ? 'disabled' : ''}>${esc(a.label)}</button>`).join('');
      $$('tbody tr', dataArea).forEach(tr => {
        const row = list.find(x => String(x[cfg.rowKey || 'id']) === tr.dataset.id);
        if (!row) return;
        const actTd = tr.querySelector('td:last-child');
        if (cfg.rowActions && actTd && !actTd.querySelector('.actions')) {
          actTd.innerHTML = `<div class="actions">${actionsHtml(row)}</div>`;
        }
      });

      // 勾选
      const selAll = $('.sel-all', dataArea);
      if (selAll) selAll.onchange = () => {
        $$('.sel-row', dataArea).forEach(cb => { cb.checked = selAll.checked; selAll.checked ? selected.add(cb.value) : selected.delete(cb.value); });
        updateBatch();
      };
      $$('.sel-row', dataArea).forEach(cb => cb.onchange = () => {
        cb.checked ? selected.add(cb.value) : selected.delete(cb.value);
        updateBatch();
      });
      function updateBatch() {
        if (!cfg.onBatchDelete) return;
        let btn = $('#batch-del', headerActions);
        if (!btn) {
          btn = htmlToEl('<button class="btn btn-danger" id="batch-del">批量删除</button>');
          headerActions.appendChild(btn);
        }
        btn.style.display = selected.size ? '' : 'none';
        btn.textContent = `批量删除(${selected.size})`;
        btn.onclick = async () => {
          if (!await confirmBox(`确定删除选中的 <b>${selected.size}</b> 项吗？`)) return;
          try { await cfg.onBatchDelete(Array.from(selected)); toast('删除成功'); selected.clear(); load(); }
          catch (e) { toastError(e); }
        };
        btn.style.display = selected.size ? '' : 'none';
      }
      updateBatch();

      // 操作按钮
      $$('[data-act]', dataArea).forEach(btn => {
        const row = list.find(x => String(x[cfg.rowKey || 'id']) === btn.dataset.id);
        const actions = cfg.rowActions ? cfg.rowActions(row) : [];
        const action = actions[Number(btn.dataset.act)];
        if (action) btn.onclick = () => action.onClick(row, load);
      });

      // 分页
      $$('.pager button', dataArea).forEach(b => b.onclick = () => {
        page = Number(b.dataset.p);
        load();
      });
    } catch (e) {
      dataArea.innerHTML = `<div class="empty">${esc(e.message || '加载失败')}</div>`;
      toastError(e);
    }
  }
  load();
  return { reload: load };
}

// ============================ 布局 / 路由 ============================
const pageTimers = [];
function registerTimer(id) { pageTimers.push(id); }
function clearPageTimers() { pageTimers.forEach(clearInterval); pageTimers.length = 0; }

function flattenMenus(menus, out) {
  out = out || [];
  (menus || []).forEach(m => {
    if (m.type === 3) return;
    out.push(m);
    if (m.children && m.children.length) flattenMenus(m.children, out);
  });
  return out;
}

function findMenuPath(menus, path) {
  for (const m of menus || []) {
    if (m.path === path) return m;
    const hit = findMenuPath(m.children, path);
    if (hit) return hit;
  }
  return null;
}

function findParentChain(menus, path, chain) {
  chain = chain || [];
  for (const m of menus || []) {
    if (m.path === path) return chain.concat([m]);
    const hit = findParentChain(m.children, path, chain.concat([m]));
    if (hit) return hit;
  }
  return null;
}

function renderShell() {
  $('#login-page').classList.add('hidden');
  $('#app').classList.remove('hidden');
  $('#brand-name').textContent = state.publicInfo.name || 'ServerAdmin';
  document.title = state.publicInfo.name || 'ServerAdmin';
  $('#brand-version').textContent = state.publicInfo.version || '';
  $('#user-nickname').textContent = state.user.nickname || state.user.username;
  $('#user-avatar').textContent = (state.user.nickname || state.user.username).slice(0, 1).toUpperCase();

  // 侧边栏
  const menu = $('#side-menu');
  menu.innerHTML = '';
  (state.menus || []).forEach(m => {
    if (m.visible === false) return;
    if (m.type === 1) {
      menu.appendChild(htmlToEl(`<div class="menu-group-title">${esc(m.icon || '')} ${esc(m.name)}</div>`));
      (m.children || []).forEach(child => {
        if (child.visible === false || child.type === 3) return;
        menu.appendChild(htmlToEl(`
          <div class="menu-item" data-path="${esc(child.path)}">
            <span class="icon">${esc(child.icon || '📄')}</span><span>${esc(child.name)}</span>
          </div>`));
      });
    } else if (m.type === 2) {
      menu.appendChild(htmlToEl(`
        <div class="menu-item" data-path="${esc(m.path)}">
          <span class="icon">${esc(m.icon || '📄')}</span><span>${esc(m.name)}</span>
        </div>`));
    }
  });
  menu.addEventListener('click', e => {
    const item = e.target.closest('.menu-item');
    if (item) location.hash = '#' + item.dataset.path;
  });

  // 用户菜单
  const chip = $('#user-chip'), drop = $('#user-dropdown');
  chip.onclick = e => {
    drop.classList.toggle('hidden');
    e.stopPropagation();
  };
  document.addEventListener('click', () => drop.classList.add('hidden'));
  drop.addEventListener('click', e => {
    const a = e.target.closest('[data-action]');
    if (!a) return;
    if (a.dataset.action === 'profile') { location.hash = '#/profile'; route(); }
    else if (a.dataset.action === 'logout') doLogout();
    drop.classList.add('hidden');
  });
}

async function doLogout() {
  try { await api('/auth/logout', { method: 'POST', body: {} }); } catch (e) { /* 忽略 */ }
  clearTokens();
  location.hash = '';
  $('#app').classList.add('hidden');
  renderLogin();
}

function route() {
  clearPageTimers();
  const path = (location.hash || '#/').slice(1) || '/';
  const page = $('#page');
  page.innerHTML = '';

  let menu = null, comp = null;
  if (path === '/profile') {
    comp = 'profile';
  } else {
    menu = findMenuPath(state.menus, path) || (path === '/' ? { component: 'dashboard', name: '首页', path: '/dashboard' } : null);
    if (menu) comp = menu.component;
  }

  // 高亮菜单
  $$('.menu-item').forEach(el => el.classList.toggle('active', el.dataset.path === (menu ? menu.path : path)));

  // 面包屑
  const bc = $('#breadcrumb');
  if (menu) {
    const chain = findParentChain(state.menus, menu.path) || [menu];
    bc.innerHTML = chain.map((m, i) =>
      i === chain.length - 1 ? `<b>${esc(m.name)}</b>` : `${esc(m.name)}<span class="sep">/</span>`).join('');
  } else {
    bc.innerHTML = `<b>${comp === 'profile' ? '个人中心' : '首页'}</b>`;
  }

  if (comp && window.Pages && window.Pages[comp]) {
    window.Pages[comp](page);
  } else {
    page.innerHTML = `<div class="card"><div class="empty" style="padding:60px;text-align:center;color:#94a3b8">
      <div style="font-size:40px;margin-bottom:10px">🚧</div>页面不存在或未授权访问</div></div>`;
  }
}

// ============================ 登录页 ============================
async function renderLogin() {
  document.title = state.publicInfo.name || 'ServerAdmin';
  $('#app').classList.add('hidden');
  const wrap = $('#login-page');
  wrap.classList.remove('hidden');
  wrap.innerHTML = `
    <div class="login-card">
      <div class="login-logo">🛡️</div>
      <div class="login-title">${esc(state.publicInfo.name || 'ServerAdmin')}</div>
      <div class="login-sub">Go + Gin + MongoDB 后台管理系统</div>
      <form id="login-form">
        <div class="form-item"><input type="text" name="username" placeholder="用户名" required></div>
        <div class="form-item"><input type="password" name="password" placeholder="密码" required></div>
        <div class="form-item login-captcha-row hidden" id="captcha-row">
          <input type="text" name="captchaCode" placeholder="验证码" maxlength="4">
          <img class="login-captcha-img" id="captcha-img" title="点击刷新" alt="验证码">
        </div>
        <button class="btn btn-primary" style="width:100%;justify-content:center;padding:11px" type="submit">登 录</button>
      </form>
      <div class="login-footer">${esc(state.publicInfo.copyright || '')}</div>
    </div>`;

  let captchaId = '';
  const captchaRow = $('#captcha-row', wrap);
  try {
    const cap = await api('/auth/captcha');
    captchaId = cap.captchaId;
    $('#captcha-img', wrap).src = cap.image;
    captchaRow.classList.remove('hidden');
    $('#captcha-img', wrap).onclick = async () => {
      const c = await api('/auth/captcha');
      captchaId = c.captchaId;
      $('#captcha-img', wrap).src = c.image;
    };
  } catch (e) { captchaRow.classList.add('hidden'); }

  $('#login-form', wrap).addEventListener('submit', async e => {
    e.preventDefault();
    const f = e.target;
    const btn = $('button[type="submit"]', f);
    btn.disabled = true;
    try {
      const res = await api('/auth/login', {
        method: 'POST',
        body: {
          username: f.username.value.trim(),
          password: f.password.value,
          captchaId,
          captchaCode: f.captchaCode ? f.captchaCode.value.trim() : '',
        },
      });
      saveTokens(res.accessToken, res.refreshToken);
      toast('登录成功，欢迎回来！');
      await bootApp();
    } catch (err) {
      toastError(err);
      // 刷新验证码
      if (captchaRow && !captchaRow.classList.contains('hidden')) {
        try {
          const c = await api('/auth/captcha');
          captchaId = c.captchaId;
          $('#captcha-img', wrap).src = c.image;
          f.captchaCode.value = '';
        } catch (e2) { /* 忽略 */ }
      }
    } finally {
      btn.disabled = false;
    }
  });
}

// ============================ 启动 ============================
async function bootApp() {
  const profile = await api('/auth/profile');
  state.user = profile.user;
  state.roles = profile.roles || [];
  state.perms = profile.perms || [];
  state.menus = profile.menus || [];
  await renderShell();
  route();
}

async function boot() {
  // 公开信息（登录页标题）
  try {
    state.publicInfo = await api('/configs/public') || state.publicInfo;
  } catch (e) { /* 使用默认 */ }

  window.addEventListener('hashchange', () => { if (state.user) route(); });

  if (!state.token) { renderLogin(); return; }
  try {
    await bootApp();
  } catch (e) {
    if (!e.silent) toastError(e);
    clearTokens();
    renderLogin();
  }
}

document.addEventListener('DOMContentLoaded', boot);
