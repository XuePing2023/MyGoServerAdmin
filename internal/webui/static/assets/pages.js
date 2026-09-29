/* 管理页面 Part1：仪表盘 / 用户 / 角色 / 菜单 / 部门 / 字典 */
'use strict';
window.Pages = window.Pages || {};

// ============================ 树工具 ============================
function flattenTree(nodes, depth, out, collapse) {
  depth = depth || 0; out = out || [];
  (nodes || []).forEach(n => {
    const kids = (n.children || []).filter(c => true);
    n.__depth = depth;
    n.__hasKids = kids.length > 0;
    n.__collapsed = !!(collapse && collapse.has(n.id));
    out.push(n);
    if (!n.__collapsed) flattenTree(kids, depth + 1, out, collapse);
  });
  return out;
}

function treeIndent(n) {
  const pad = '&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(n.__depth);
  const toggle = n.__hasKids
    ? `<span class="tree-toggle" data-toggle="${esc(n.id)}">${n.__collapsed ? '▸' : '▾'}</span>`
    : '<span class="tree-toggle"></span>';
  return `${pad}${toggle}`;
}

// 收集选中节点的所有祖先 id（保证勾选父级）
function collectAncestors(tree, selectedSet) {
  const parentOf = {};
  const walk = (nodes, parent) => (nodes || []).forEach(n => {
    parentOf[n.id] = parent;
    walk(n.children, n.id);
  });
  walk(tree, '');
  const result = new Set(selectedSet);
  selectedSet.forEach(id => {
    let p = parentOf[id];
    while (p) { result.add(p); p = parentOf[p]; }
  });
  return Array.from(result);
}

function treeCheckboxHtml(nodes, depth, selectedSet) {
  return (nodes || []).map(n => {
    const pad = '&nbsp;&nbsp;&nbsp;&nbsp;'.repeat(depth);
    const self = `<div class="tree-check-item">${pad}<label style="display:flex;gap:6px;align-items:center">
      <input type="checkbox" value="${esc(n.id)}" data-treecheck ${selectedSet.has(n.id) ? 'checked' : ''}>
      <span>${esc(n.icon && n.type !== 3 ? n.icon + ' ' : '')}${esc(n.name)}${n.type === 3 ? ' <span class="dim">『按钮』</span>' : ''}</span></label></div>`;
    return self + treeCheckboxHtml(n.children, depth + 1, selectedSet);
  }).join('');
}

// ============================ 首页仪表盘 ============================
window.Pages.dashboard = async function (container) {
  container.innerHTML = '<div class="empty" style="padding:40px;color:#94a3b8">加载中...</div>';
  let st;
  try { st = await api('/dashboard/stats'); } catch (e) { container.innerHTML = `<div class="card"><div class="empty">${esc(e.message)}</div></div>`; return; }

  const cards = [
    ['👥', '用户总数', st.userCount, '#eef1fe'],
    ['🎭', '角色数量', st.roleCount, '#fdf2f8'],
    ['🟢', '在线用户', st.onlineCount, '#ecfdf5'],
    ['📥', '今日登录', st.todayLogins, '#fffbeb'],
    ['📝', '今日操作', st.todayOpLogs, '#eff6ff'],
    ['⏰', '运行中任务', st.jobEnabledCount, '#f5f3ff'],
  ];
  const maxTrend = Math.max(...st.trend.map(t => t.count), 1);

  container.innerHTML = `
    <div class="page-header"><div>
      <div class="page-title">仪表盘</div>
      <div class="page-desc">系统运行概况（数据实时统计）</div>
    </div></div>
    <div class="stat-grid">
      ${cards.map(([icon, label, num, bg]) => `
        <div class="stat-card">
          <div class="stat-icon" style="background:${bg}">${icon}</div>
          <div><div class="stat-num">${num}</div><div class="stat-label">${label}</div></div>
        </div>`).join('')}
    </div>
    <div class="card">
      <div class="card-title">近 7 天登录趋势</div>
      <div class="trend-chart">
        ${st.trend.map(t => `
          <div class="trend-col" title="${t.date}：${t.count} 次">
            <div class="trend-val">${t.count}</div>
            <div class="trend-bar" style="height:${Math.max(t.count / maxTrend * 100, 2)}%"></div>
            <div class="trend-date">${t.date.slice(5)}</div>
          </div>`).join('')}
      </div>
    </div>
    <div class="card">
      <div class="card-title">最近操作</div>
      ${renderTable([
        { title: '时间', key: 'createdAt', render: r => `<span class="dim">${fmtTime(r.createdAt)}</span>` },
        { title: '操作人', key: 'username' },
        { title: '模块', key: 'module' },
        { title: '动作', key: 'action' },
        { title: '路径', render: r => `<span class="mono">${esc(r.method)} ${esc(r.path)}</span>` },
        { title: '结果', render: r => r.bizCode === 0 ? tag('成功', 'success') : tag('失败', 'danger') },
      ], st.recentOpLogs || [])}
    </div>
    <div class="card">
      <div class="card-title">最新公告</div>
      <ul class="notice-list">
        ${(st.recentNotices || []).map(n => `
          <li>
            ${n.type === 2 ? tag('公告', 'warning') : tag('通知', 'info')}
            <span class="title">${esc(n.title)}</span>
            <span class="dim">${esc(n.publisher)}</span>
            <span class="dim">${fmtTime(n.createdAt)}</span>
          </li>`).join('') || '<li class="dim">暂无公告</li>'}
      </ul>
    </div>`;
};

// ============================ 用户管理 ============================
async function deptOptions() {
  const tree = await api('/departments/tree');
  const opts = [{ label: '未分配', value: '' }];
  const walk = (nodes, depth) => (nodes || []).forEach(n => {
    opts.push({ label: '　'.repeat(depth) + n.name, value: n.id });
    walk(n.children, depth + 1);
  });
  walk(tree, 0);
  return opts;
}

async function roleOptions() {
  const roles = await api('/roles', { query: { all: 1 } });
  return (roles.list || roles || []).map(r => ({ label: r.name, value: r.code }));
}

function userFormFields(edit, extra) {
  const f = [
    { key: 'username', label: '用户名', required: !edit, disabled: !!edit, help: edit ? '用户名创建后不可修改' : '字母数字_-@.，2-32 位' },
    { key: 'nickname', label: '昵称', required: true },
  ];
  if (!edit) f.push({ key: 'password', label: '初始密码', type: 'password', help: '留空则使用默认密码 admin123' });
  f.push(
    { key: 'email', label: '邮箱', type: 'email' },
    { key: 'phone', label: '手机号' },
    { key: 'gender', label: '性别', type: 'select', options: extra.genders },
    { key: 'deptId', label: '所属部门', type: 'select', options: extra.depts },
    { key: 'roles', label: '角色', type: 'checkboxes', options: extra.roles },
    { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }] },
    { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
  );
  return f;
}

window.Pages.users = function (container) {
  const can = {
    create: hasPerm('sys:user:create'), update: hasPerm('sys:user:update'),
    del: hasPerm('sys:user:delete'), reset: hasPerm('sys:user:resetpwd'), status: hasPerm('sys:user:status'),
  };
  listPage(container, {
    title: '用户管理', desc: '管理后台账号、角色分配与状态',
    filters: [
      { key: 'username', label: '用户名' },
      { key: 'nickname', label: '昵称' },
      { key: 'status', label: '状态', type: 'select', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }] },
    ],
    loadData: q => api('/users', { query: q }),
    canAdd: can.create,
    onAdd: async reload => {
      const [depts, roles, genders] = await Promise.all([deptOptions(), roleOptions(), dict('sys_gender')]);
      formModal({
        title: '新增用户',
        values: { status: 1 },
        fields: userFormFields(false, { depts, roles, genders: genders.map(g => ({ label: g.label, value: Number(g.value) })) }),
        onSubmit: async v => {
          await api('/users', { method: 'POST', body: v });
          toast('创建成功'); reload();
        },
      });
    },
    checkbox: can.del,
    onBatchDelete: ids => api('/users', { method: 'DELETE', body: { ids } }),
    columns: [
      { title: '用户名', key: 'username', render: r => `<b>${esc(r.username)}</b>` },
      { title: '昵称', key: 'nickname' },
      { title: '部门', key: 'deptName' },
      { title: '角色', render: r => (r.roleNames || []).map(n => tag(n, 'primary')).join(' ') || '<span class="dim">-</span>' },
      { title: '手机号', key: 'phone' },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '最后登录', render: r => `<span class="dim">${fmtTime(r.lastLogin)}</span>` },
      { title: '操作', width: '220px' },
    ],
    rowActions: r => {
      const acts = [];
      if (can.update) acts.push({ label: '编辑', onClick: async (row, reload) => {
        const [depts, roles, genders] = await Promise.all([deptOptions(), roleOptions(), dict('sys_gender')]);
        formModal({
          title: '编辑用户：' + row.username,
          fields: userFormFields(true, { depts, roles, genders: genders.map(g => ({ label: g.label, value: Number(g.value) })) }),
          values: row,
          onSubmit: async v => {
            delete v.username; delete v.password;
            await api('/users/' + row.id, { method: 'PUT', body: v });
            toast('保存成功'); reload();
          },
        });
      } });
      if (can.reset) acts.push({ label: '重置密码', onClick: async row => {
        formModal({
          title: '重置密码：' + row.username, width: 'modal-sm',
          fields: [{ key: 'password', label: '新密码', type: 'password', required: true, help: '至少 6 位' }],
          onSubmit: async v => {
            await api(`/users/${row.id}/password`, { method: 'PUT', body: v });
            toast('密码已重置');
          },
        });
      } });
      if (can.status) acts.push({ label: r.status === 1 ? '禁用' : '启用', onClick: async (row, reload) => {
        await api(`/users/${row.id}/status`, { method: 'PUT', body: { status: r.status === 1 ? 2 : 1 } });
        toast('操作成功'); reload();
      } });
      if (can.del) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除用户 <b>${esc(row.username)}</b> 吗？`)) return;
        await api('/users', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });
};

// ============================ 角色管理 ============================
window.Pages.roles = function (container) {
  const can = {
    create: hasPerm('sys:role:create'), update: hasPerm('sys:role:update'),
    del: hasPerm('sys:role:delete'), assign: hasPerm('sys:role:assign'),
  };
  listPage(container, {
    title: '角色管理', desc: 'RBAC 角色与菜单/按钮权限分配',
    filters: [{ key: 'name', label: '角色名称' }],
    loadData: q => api('/roles', { query: q }),
    canAdd: can.create,
    onAdd: reload => formModal({
      title: '新增角色',
      values: { status: 1, sort: 0 },
      fields: [
        { key: 'name', label: '角色名称', required: true },
        { key: 'code', label: '角色编码', required: true, help: '字母开头，如 ops / viewer' },
        { key: 'sort', label: '排序', type: 'number', value: 0 },
        { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }] },
        { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
      ],
      onSubmit: async v => { await api('/roles', { method: 'POST', body: v }); toast('创建成功'); reload(); },
    }),
    columns: [
      { title: '角色名称', key: 'name', render: r => `<b>${esc(r.name)}</b>${r.builtIn ? ' ' + tag('内置', 'warning') : ''}` },
      { title: '编码', render: r => `<span class="mono">${esc(r.code)}</span>` },
      { title: '排序', key: 'sort' },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '备注', key: 'remark' },
      { title: '创建时间', render: r => `<span class="dim">${fmtTime(r.createdAt)}</span>` },
      { title: '操作', width: '240px' },
    ],
    rowActions: r => {
      const acts = [];
      if (can.update) acts.push({ label: '编辑', onClick: (row, reload) => formModal({
        title: '编辑角色',
        fields: [
          { key: 'name', label: '角色名称', required: true },
          { key: 'code', label: '角色编码', required: true, disabled: !!row.builtIn },
          { key: 'sort', label: '排序', type: 'number' },
          { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], disabled: !!row.builtIn },
          { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
        ],
        values: row,
        onSubmit: async v => {
          if (row.builtIn) { v.code = row.code; v.status = row.status; }
          await api('/roles/' + row.id, { method: 'PUT', body: v });
          toast('保存成功'); reload();
        },
      }) });
      if (can.assign && !r.builtIn) acts.push({ label: '分配权限', onClick: async row => {
        const tree = await api('/menus/tree');
        const selected = new Set(row.menus || []);
        const box = htmlToEl(`<div class="tree-check">${treeCheckboxHtml(tree, 0, selected)}</div>`);
        const m = openModal({
          title: `分配权限：${row.name}`,
          width: 'modal-lg',
          body: `<div class="form-help mb8">勾选该角色可访问的菜单与按钮权限（父级自动勾选）</div>`,
          footer: (() => {
            const f = htmlToEl(`<div style="display:flex;gap:10px">
              <button class="btn" data-act="cancel">取消</button>
              <button class="btn btn-primary" data-act="ok">保 存</button></div>`);
            f.querySelector('[data-act="cancel"]').onclick = () => m.close();
            f.querySelector('[data-act="ok"]').onclick = async () => {
              const ids = $$('[data-treecheck]', box).filter(i => i.checked).map(i => i.value);
              try {
                await api(`/roles/${row.id}/menus`, { method: 'PUT', body: { menuIds: collectAncestors(tree, new Set(ids)) } });
                toast('权限已更新'); m.close(); window.Pages.roles($('#page'));
              } catch (e) { toastError(e); }
            };
            return f;
          })(),
        });
        $('.modal-body', m).appendChild(box);
      } });
      if (can.del && !r.builtIn) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除角色 <b>${esc(row.name)}</b> 吗？`)) return;
        await api('/roles', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });
};

// ============================ 菜单管理 ============================
window.Pages.menus = function (container) {
  const can = {
    create: hasPerm('sys:menu:create'), update: hasPerm('sys:menu:update'), del: hasPerm('sys:menu:delete'),
  };
  const collapse = new Set();
  const view = htmlToEl(`
    <div>
      <div class="page-header">
        <div><div class="page-title">菜单管理</div>
        <div class="page-desc">前端路由菜单与按钮级权限标识（类型：目录 / 菜单 / 按钮）</div></div>
        ${can.create ? '<button class="btn btn-primary" id="add-menu">＋ 新增菜单</button>' : ''}
      </div>
      <div class="card"><div class="data-area">加载中...</div></div>
    </div>`);
  container.appendChild(view);

  async function load() {
    const tree = await api('/menus/tree');
    const rows = flattenTree(tree, 0, [], collapse);
    $('.data-area', view).innerHTML = renderTable([
      { title: '名称', render: r => `${treeIndent(r)}${esc(r.icon || '')} <b>${esc(r.name)}</b>` },
      { title: '类型', render: r => ['', tag('目录', 'primary'), tag('菜单', 'success'), tag('按钮', 'warning')][r.type] },
      { title: '路由', render: r => r.path ? `<span class="mono">${esc(r.path)}</span>` : '-' },
      { title: '组件', render: r => r.component ? `<span class="mono">${esc(r.component)}</span>` : '-' },
      { title: '权限标识', render: r => r.perm ? `<span class="mono">${esc(r.perm)}</span>` : '-' },
      { title: '排序', key: 'sort' },
      { title: '可见', render: r => r.visible ? '是' : '否' },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '操作', width: '200px' },
    ], rows);

    // 展开/收起
    $$('[data-toggle]', view).forEach(el => el.onclick = () => {
      const id = el.dataset.toggle;
      collapse.has(id) ? collapse.delete(id) : collapse.add(id);
      load();
    });

    // 操作
    $$('tbody tr', view).forEach(tr => {
      const row = rows.find(x => x.id === tr.dataset.id);
      if (!row) return;
      const td = tr.querySelector('td:last-child');
      td.innerHTML = `<div class="actions"></div>`;
      const box = td.querySelector('.actions');
      const mkBtn = (label, fn, danger) => {
        const b = htmlToEl(`<button class="btn-link ${danger ? 'danger' : ''}">${label}</button>`);
        b.onclick = fn; box.appendChild(b);
      };
      if (can.create) mkBtn('新增子项', () => openForm(row, null));
      if (can.update) mkBtn('编辑', () => openForm(null, row));
      if (can.del) mkBtn('删除', async () => {
        if (!await confirmBox(`确定删除菜单 <b>${esc(row.name)}</b> 吗？`)) return;
        try { await api('/menus/' + row.id, { method: 'DELETE' }); toast('删除成功'); load(); }
        catch (e) { toastError(e); }
      }, true);
    });
  }

  function openForm(parent, row) {
    api('/menus/tree').then(tree => {
      const opts = [{ label: '根目录', value: '' }];
      flattenTree(tree, 0, []).forEach(n => opts.push({ label: '　'.repeat(n.__depth) + n.name, value: n.id }));
      formModal({
        title: row ? '编辑菜单：' + row.name : (parent ? `新增子菜单（父级：${parent.name}）` : '新增菜单'),
        width: 'modal-lg',
        values: row ? Object.assign({}, row, { parentId: row.parentId }) : { parentId: parent ? parent.id : '', visible: true, status: 1, sort: 0 },
        fields: [
          { key: 'parentId', label: '上级菜单', type: 'select', options: opts },
          { key: 'type', label: '类型', type: 'select', options: [{ label: '目录', value: 1 }, { label: '菜单', value: 2 }, { label: '按钮', value: 3 }], required: true },
          { key: 'name', label: '名称', required: true },
          { key: 'path', label: '路由路径', placeholder: '/system/users' },
          { key: 'component', label: '页面组件', placeholder: 'users（对应前端页面标识）' },
          { key: 'perm', label: '权限标识', placeholder: 'sys:user:create' },
          { key: 'icon', label: '图标', placeholder: 'emoji 或留空' },
          { key: 'sort', label: '排序', type: 'number' },
          { key: 'visible', label: '是否可见', type: 'switch' },
          { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
        ],
        onSubmit: async v => {
          v.type = Number(v.type); v.status = Number(v.status); v.visible = !!v.visible; v.sort = Number(v.sort || 0);
          if (row) { await api('/menus/' + row.id, { method: 'PUT', body: v }); toast('保存成功'); }
          else { await api('/menus', { method: 'POST', body: v }); toast('创建成功'); }
          load();
        },
      });
    }).catch(toastError);
  }

  const addBtn = $('#add-menu', view);
  if (addBtn) addBtn.onclick = () => openForm(null, null);
  load().catch(toastError);
};

// ============================ 部门管理 ============================
window.Pages.depts = function (container) {
  const can = {
    create: hasPerm('sys:dept:create'), update: hasPerm('sys:dept:update'), del: hasPerm('sys:dept:delete'),
  };
  const collapse = new Set();
  const view = htmlToEl(`
    <div>
      <div class="page-header">
        <div><div class="page-title">部门管理</div><div class="page-desc">组织架构（树形结构）</div></div>
        ${can.create ? '<button class="btn btn-primary" id="add-dept">＋ 新增部门</button>' : ''}
      </div>
      <div class="card"><div class="data-area">加载中...</div></div>
    </div>`);
  container.appendChild(view);

  async function load() {
    const tree = await api('/departments/tree');
    const rows = flattenTree(tree, 0, [], collapse);
    $('.data-area', view).innerHTML = renderTable([
      { title: '部门名称', render: r => `${treeIndent(r)}<b>${esc(r.name)}</b>` },
      { title: '负责人', key: 'leader' },
      { title: '联系电话', key: 'phone' },
      { title: '排序', key: 'sort' },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '操作', width: '180px' },
    ], rows);

    $$('[data-toggle]', view).forEach(el => el.onclick = () => {
      const id = el.dataset.toggle;
      collapse.has(id) ? collapse.delete(id) : collapse.add(id);
      load();
    });

    $$('tbody tr', view).forEach(tr => {
      const row = rows.find(x => x.id === tr.dataset.id);
      if (!row) return;
      const td = tr.querySelector('td:last-child');
      td.innerHTML = `<div class="actions"></div>`;
      const box = td.querySelector('.actions');
      const mkBtn = (label, fn, danger) => {
        const b = htmlToEl(`<button class="btn-link ${danger ? 'danger' : ''}">${label}</button>`);
        b.onclick = fn; box.appendChild(b);
      };
      if (can.update) mkBtn('编辑', async () => {
        const tree2 = await api('/departments/tree');
        const opts = [{ label: '顶级部门', value: '' }];
        flattenTree(tree2, 0, []).forEach(n => { if (n.id !== row.id) opts.push({ label: '　'.repeat(n.__depth) + n.name, value: n.id }); });
        formModal({
          title: '编辑部门：' + row.name,
          values: row,
          fields: [
            { key: 'parentId', label: '上级部门', type: 'select', options: opts },
            { key: 'name', label: '部门名称', required: true },
            { key: 'leader', label: '负责人' },
            { key: 'phone', label: '联系电话' },
            { key: 'sort', label: '排序', type: 'number' },
            { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
          ],
          onSubmit: async v => {
            await api('/departments/' + row.id, { method: 'PUT', body: v });
            toast('保存成功'); load();
          },
        });
      });
      if (can.create) mkBtn('新增子部门', () => openForm(row));
      if (can.del) mkBtn('删除', async () => {
        if (!await confirmBox(`确定删除部门 <b>${esc(row.name)}</b> 吗？`)) return;
        try { await api('/departments/' + row.id, { method: 'DELETE' }); toast('删除成功'); load(); }
        catch (e) { toastError(e); }
      }, true);
    });
  }

  async function openForm(parent) {
    const tree = await api('/departments/tree');
    const opts = [{ label: '顶级部门', value: '' }];
    flattenTree(tree, 0, []).forEach(n => opts.push({ label: '　'.repeat(n.__depth) + n.name, value: n.id }));
    formModal({
      title: parent ? `新增子部门（父级：${parent.name}）` : '新增部门',
      values: { parentId: parent ? parent.id : '', status: 1, sort: 0 },
      fields: [
        { key: 'parentId', label: '上级部门', type: 'select', options: opts },
        { key: 'name', label: '部门名称', required: true },
        { key: 'leader', label: '负责人' },
        { key: 'phone', label: '联系电话' },
        { key: 'sort', label: '排序', type: 'number' },
        { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
      ],
      onSubmit: async v => {
        await api('/departments', { method: 'POST', body: v });
        toast('创建成功'); load();
      },
    });
  }

  const addBtn = $('#add-dept', view);
  if (addBtn) addBtn.onclick = () => openForm(null);
  load().catch(toastError);
};

// ============================ 字典管理 ============================
window.Pages.dicts = function (container) {
  const can = {
    create: hasPerm('sys:dict:create'), update: hasPerm('sys:dict:update'), del: hasPerm('sys:dict:delete'),
  };
  let currentType = '';

  container.innerHTML = `
    <div class="page-header">
      <div><div class="page-title">字典管理</div><div class="page-desc">系统枚举值维护（点击左侧类型查看字典项）</div></div>
    </div>
    <div class="split">
      <div class="card">
        <div class="toolbar">
          <div class="card-title" style="margin:0">字典类型</div>
          <span class="spacer"></span>
          ${can.create ? '<button class="btn btn-primary btn-sm" id="add-type">＋ 新增</button>' : ''}
        </div>
        <div id="type-area">加载中...</div>
      </div>
      <div class="card">
        <div class="toolbar">
          <div class="card-title" style="margin:0">字典项 <span class="dim" id="item-type-name"></span></div>
          <span class="spacer"></span>
          ${can.create ? '<button class="btn btn-primary btn-sm" id="add-item">＋ 新增字典项</button>' : ''}
        </div>
        <div id="item-area"><div class="empty">请先在左侧选择字典类型</div></div>
      </div>
    </div>`;

  async function loadTypes() {
    const res = await api('/dict-types', { query: { page: 1, size: 100 } });
    const list = res.list || [];
    $('#type-area').innerHTML = renderTable([
      { title: '名称', render: r => `<b>${esc(r.name)}</b>` },
      { title: '编码', render: r => `<span class="mono">${esc(r.code)}</span>` },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '操作', render: r => `<div class="actions"></div>` },
    ], list);
    $$('#type-area tbody tr').forEach(tr => {
      const row = list.find(x => x.id === tr.dataset.id);
      if (!row) return;
      tr.classList.add('clickable-row');
      if (row.code === currentType) tr.classList.add('active');
      tr.onclick = e => { if (e.target.closest('button')) return; currentType = row.code; loadTypes(); loadItems(); };
      const td = tr.querySelector('td:last-child');
      td.innerHTML = `<div class="actions"></div>`;
      const box = td.querySelector('.actions');
      const mkBtn = (label, fn, danger) => {
        const b = htmlToEl(`<button class="btn-link ${danger ? 'danger' : ''}">${label}</button>`);
        b.onclick = fn; box.appendChild(b);
      };
      if (can.update) mkBtn('编辑', () => formModal({
        title: '编辑字典类型', values: row,
        fields: [
          { key: 'name', label: '名称', required: true },
          { key: 'code', label: '编码', required: true },
          { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
          { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
        ],
        onSubmit: async v => { await api('/dict-types/' + row.id, { method: 'PUT', body: v }); toast('保存成功'); loadTypes(); },
      }));
      if (can.del) mkBtn('删除', async () => {
        if (!await confirmBox(`确定删除字典 <b>${esc(row.name)}</b> 及其全部字典项吗？`)) return;
        try { await api('/dict-types', { method: 'DELETE', body: { ids: [row.id] } }); toast('删除成功'); if (currentType === row.code) { currentType = ''; loadItems(); } loadTypes(); }
        catch (e) { toastError(e); }
      }, true);
    });
  }

  async function loadItems() {
    $('#item-type-name').textContent = currentType ? `（${currentType}）` : '';
    if (!currentType) { $('#item-area').innerHTML = '<div class="empty">请先在左侧选择字典类型</div>'; return; }
    const res = await api('/dict-items', { query: { typeCode: currentType, page: 1, size: 100 } });
    const list = res.list || [];
    $('#item-area').innerHTML = renderTable([
      { title: '标签', key: 'label' },
      { title: '键值', render: r => `<span class="mono">${esc(r.value)}</span>` },
      { title: '标签色', render: r => r.tagType ? tag(r.tagType, r.tagType) : '-' },
      { title: '排序', key: 'sort' },
      { title: '状态', render: r => statusTag(r.status) },
      { title: '操作', render: () => `<div class="actions"></div>` },
    ], list);
    $$('#item-area tbody tr').forEach(tr => {
      const row = list.find(x => x.id === tr.dataset.id);
      if (!row) return;
      const td = tr.querySelector('td:last-child');
      td.innerHTML = `<div class="actions"></div>`;
      const box = td.querySelector('.actions');
      const mkBtn = (label, fn, danger) => {
        const b = htmlToEl(`<button class="btn-link ${danger ? 'danger' : ''}">${label}</button>`);
        b.onclick = fn; box.appendChild(b);
      };
      if (can.update) mkBtn('编辑', () => formModal({
        title: '编辑字典项', values: row,
        fields: [
          { key: 'label', label: '标签', required: true },
          { key: 'value', label: '键值', required: true },
          { key: 'tagType', label: '标签色', type: 'select', options: [{ label: '无', value: '' }, { label: 'success', value: 'success' }, { label: 'info', value: 'info' }, { label: 'warning', value: 'warning' }, { label: 'danger', value: 'danger' }] },
          { key: 'sort', label: '排序', type: 'number' },
          { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
          { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
        ],
        onSubmit: async v => { await api('/dict-items/' + row.id, { method: 'PUT', body: Object.assign({ typeCode: currentType }, v) }); toast('保存成功'); delete dictCache[currentType]; loadItems(); },
      }));
      if (can.del) mkBtn('删除', async () => {
        if (!await confirmBox(`确定删除字典项 <b>${esc(row.label)}</b> 吗？`)) return;
        try { await api('/dict-items', { method: 'DELETE', body: { ids: [row.id] } }); toast('删除成功'); loadItems(); }
        catch (e) { toastError(e); }
      }, true);
    });
  }

  const addType = $('#add-type');
  if (addType) addType.onclick = () => formModal({
    title: '新增字典类型',
    values: { status: 1 },
    fields: [
      { key: 'name', label: '名称', required: true },
      { key: 'code', label: '编码', required: true, placeholder: '如 sys_yes_no' },
      { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
      { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
    ],
    onSubmit: async v => { await api('/dict-types', { method: 'POST', body: v }); toast('创建成功'); loadTypes(); },
  });

  const addItem = $('#add-item');
  if (addItem) addItem.onclick = () => {
    if (!currentType) { toast('请先在左侧选择字典类型', 'error'); return; }
    formModal({
      title: `新增字典项（${currentType}）`,
      values: { status: 1, sort: 0 },
      fields: [
        { key: 'label', label: '标签', required: true },
        { key: 'value', label: '键值', required: true },
        { key: 'tagType', label: '标签色', type: 'select', options: [{ label: '无', value: '' }, { label: 'success', value: 'success' }, { label: 'info', value: 'info' }, { label: 'warning', value: 'warning' }, { label: 'danger', value: 'danger' }] },
        { key: 'sort', label: '排序', type: 'number', value: 0 },
        { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
      ],
      onSubmit: async v => {
        await api('/dict-items', { method: 'POST', body: Object.assign({ typeCode: currentType }, v) });
        toast('创建成功'); delete dictCache[currentType]; loadItems();
      },
    });
  };

  loadTypes().catch(toastError);
};
