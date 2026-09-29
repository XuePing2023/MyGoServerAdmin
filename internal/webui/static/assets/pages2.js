/* 管理页面 Part2：参数配置 / 操作日志 / 登录日志 / 在线用户 / 定时任务 / 通知公告 / 文件 / 监控 / 个人中心 */
'use strict';
window.Pages = window.Pages || {};

// ============================ 参数配置 ============================
window.Pages.configs = function (container) {
  const can = {
    create: hasPerm('sys:config:create'), update: hasPerm('sys:config:update'), del: hasPerm('sys:config:delete'),
  };
  listPage(container, {
    title: '参数配置', desc: '系统运行参数（Key-Value），内置参数仅允许修改值',
    filters: [{ key: 'keyword', label: '键名/名称' }],
    loadData: q => api('/configs', { query: q }),
    canAdd: can.create,
    onAdd: reload => formModal({
      title: '新增参数',
      fields: [
        { key: 'key', label: '参数键', required: true, placeholder: '如 sys.mail.host' },
        { key: 'name', label: '参数名称', required: true },
        { key: 'value', label: '参数值', type: 'textarea', rows: 2 },
        { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
      ],
      onSubmit: async v => { await api('/configs', { method: 'POST', body: v }); toast('创建成功'); reload(); },
    }),
    columns: [
      { title: '参数键', render: r => `<span class="mono">${esc(r.key)}</span>${r.builtIn ? ' ' + tag('内置', 'warning') : ''}` },
      { title: '名称', key: 'name' },
      { title: '参数值', render: r => `<span class="mono">${esc(r.value || '-')}</span>` },
      { title: '备注', key: 'remark' },
      { title: '操作', width: '140px' },
    ],
    rowActions: r => {
      const acts = [];
      if (can.update) acts.push({ label: '编辑', onClick: (row, reload) => formModal({
        title: '编辑参数' + (row.builtIn ? '（内置，仅可修改值/备注）' : ''),
        values: row,
        fields: [
          { key: 'key', label: '参数键', required: true, disabled: !!row.builtIn },
          { key: 'name', label: '参数名称', required: true, disabled: !!row.builtIn },
          { key: 'value', label: '参数值', type: 'textarea', rows: 2 },
          { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
        ],
        onSubmit: async v => {
          if (row.builtIn) { v.key = row.key; v.name = row.name; }
          await api('/configs/' + row.id, { method: 'PUT', body: v });
          toast('保存成功'); reload();
        },
      }) });
      if (can.del && !r.builtIn) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除参数 <b>${esc(r.key)}</b> 吗？`)) return;
        await api('/configs', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });
};

// ============================ 操作日志 ============================
window.Pages.oplogs = function (container) {
  const canDel = hasPerm('sys:oplog:delete');
  listPage(container, {
    title: '操作日志', desc: '记录所有非查询类请求（敏感字段已脱敏）',
    filters: [
      { key: 'username', label: '操作人' },
      { key: 'module', label: '模块' },
      { key: 'start', label: '开始日期', type: 'date' },
      { key: 'end', label: '结束日期', type: 'date' },
    ],
    columns: [
      { title: '时间', render: r => `<span class="dim">${fmtTime(r.createdAt)}</span>` },
      { title: '操作人', key: 'username' },
      { title: '模块', key: 'module' },
      { title: '动作', key: 'action' },
      { title: '请求', render: r => `<span class="mono">${esc(r.method)} ${esc(r.path)}</span>` },
      { title: 'IP', render: r => `<span class="mono">${esc(r.ip)}</span>` },
      { title: '结果', render: r => r.bizCode === 0 ? tag('成功', 'success') : tag(r.bizCode || '异常', 'danger') },
      { title: '耗时', render: r => `${r.costMs}ms` },
      { title: '操作', width: '90px' },
    ],
    rowActions: () => [{
      label: '详情', onClick: async row => {
        openModal({
          title: `操作详情：${row.module} / ${row.action}`,
          width: 'modal-lg',
          body: `
            <table class="table">
              <tr><th style="width:110px">操作人</th><td>${esc(row.username)}（${esc(row.ip)}）</td></tr>
              <tr><th>请求</th><td><span class="mono">${esc(row.method)} ${esc(row.path)}${row.query ? '?' + esc(row.query) : ''}</span></td></tr>
              <tr><th>UA</th><td class="dim" style="white-space:normal">${esc(row.userAgent || '-')}</td></tr>
              <tr><th>HTTP / 业务码</th><td>${row.httpCode} / ${row.bizCode}</td></tr>
              <tr><th>耗时</th><td>${row.costMs}ms</td></tr>
              <tr><th>请求体</th><td><pre class="body-pre">${esc(row.body || '(无)')}</pre></td></tr>
            </table>`,
          footer: '<button class="btn" onclick="this.closest(\'.modal-mask\').remove()">关 闭</button>',
        });
      },
    }],
    extraActions: (bar, reload) => {
      if (!canDel) return;
      const btn = htmlToEl('<button class="btn btn-danger">清空日志</button>');
      bar.appendChild(btn);
      btn.onclick = async () => {
        if (!await confirmBox('确定<b style="color:#ef4444">清空全部操作日志</b>吗？此操作不可恢复！')) return;
        try { await api('/oplogs', { method: 'DELETE', body: {} }); toast('已清空'); reload(); }
        catch (e) { toastError(e); }
      };
    },
  });
};

// ============================ 登录日志 ============================
window.Pages.loginlogs = function (container) {
  const canDel = hasPerm('sys:loginlog:delete');
  listPage(container, {
    title: '登录日志', desc: '记录登录成功与失败事件',
    filters: [
      { key: 'username', label: '用户名' },
      { key: 'status', label: '状态', type: 'select', options: [{ label: '成功', value: 1 }, { label: '失败', value: 2 }] },
      { key: 'start', label: '开始日期', type: 'date' },
      { key: 'end', label: '结束日期', type: 'date' },
    ],
    columns: [
      { title: '时间', render: r => `<span class="dim">${fmtTime(r.loginAt)}</span>` },
      { title: '用户名', key: 'username' },
      { title: '结果', render: r => r.status === 1 ? tag('成功', 'success') : tag('失败', 'danger') },
      { title: '说明', key: 'msg' },
      { title: 'IP', render: r => `<span class="mono">${esc(r.ip)}</span>` },
      { title: 'UserAgent', render: r => `<span class="dim" style="max-width:280px;display:inline-block;overflow:hidden;text-overflow:ellipsis;vertical-align:middle">${esc(r.userAgent || '-')}</span>` },
    ],
    extraActions: (bar, reload) => {
      if (!canDel) return;
      const btn = htmlToEl('<button class="btn btn-danger">清空日志</button>');
      bar.appendChild(btn);
      btn.onclick = async () => {
        if (!await confirmBox('确定<b style="color:#ef4444">清空全部登录日志</b>吗？')) return;
        try { await api('/loginlogs', { method: 'DELETE', body: {} }); toast('已清空'); reload(); }
        catch (e) { toastError(e); }
      };
    },
  });
};

// ============================ 在线用户 ============================
window.Pages.online = function (container) {
  listPage(container, {
    title: '在线用户', desc: '当前活跃会话（基于内存，重启后重置）',
    loadData: async () => {
      const list = await api('/online');
      return { list: list || [], total: (list || []).length };
    },
    columns: [
      { title: '用户名', render: r => `<b>${esc(r.username)}</b>` },
      { title: '昵称', key: 'nickname' },
      { title: 'IP', render: r => `<span class="mono">${esc(r.ip)}</span>` },
      { title: '登录时间', render: r => `<span class="dim">${fmtTime(r.loginAt)}</span>` },
      { title: '最后活跃', render: r => `<span class="dim">${fmtTime(r.lastSeenAt)}</span>` },
      { title: '操作', width: '120px' },
    ],
    rowActions: () => [{
      label: '强制下线', danger: true, perm: 'sys:online:kick', onClick: async (row, reload) => {
        if (!await confirmBox(`确定将用户 <b>${esc(row.username)}</b> 强制下线吗？`)) return;
        await api('/online/' + row.jti, { method: 'DELETE' });
        toast('已强制下线'); reload();
      },
    }],
  });
};

// ============================ 定时任务 ============================
window.Pages.jobs = function (container) {
  const can = {
    create: hasPerm('sys:job:create'), update: hasPerm('sys:job:update'),
    del: hasPerm('sys:job:delete'), run: hasPerm('sys:job:run'),
  };
  listPage(container, {
    title: '定时任务', desc: '基于 cron 表达式的任务调度（robfig/cron，5 段表达式）',
    filters: [{ key: 'name', label: '任务名称' }],
    loadData: q => api('/jobs', { query: q }),
    canAdd: can.create,
    onAdd: reload => jobForm(null, reload),
    columns: [
      { title: '任务名称', render: r => `<b>${esc(r.name)}</b>` },
      { title: '处理器', render: r => `<span class="mono">${esc(r.handler)}</span>` },
      { title: 'cron', render: r => `<span class="mono">${esc(r.spec)}</span>` },
      { title: '状态', render: r => statusTag(r.status, '启用', '停用') },
      { title: '执行/失败', render: r => `${r.runCount} / ${r.failCount}` },
      { title: '上次执行', render: r => `<span class="dim">${fmtTime(r.lastRun)}</span>` },
      { title: '下次执行', render: r => `<span class="dim">${fmtTime(r.nextRun)}</span>` },
      { title: '操作', width: '260px' },
    ],
    rowActions: r => {
      const acts = [];
      if (can.run) acts.push({ label: '执行一次', onClick: async (row, reload) => {
        await api(`/jobs/${row.id}/run`, { method: 'PUT' });
        toast('已触发执行，稍后可查看任务日志'); setTimeout(reload, 600);
      } });
      if (can.run) acts.push({ label: r.status === 1 ? '停用' : '启用', onClick: async (row, reload) => {
        await api(`/jobs/${row.id}/status`, { method: 'PUT', body: { status: r.status === 1 ? 2 : 1 } });
        toast('操作成功'); reload();
      } });
      acts.push({ label: '日志', onClick: row => showJobLogs(row) });
      if (can.update) acts.push({ label: '编辑', onClick: (row, reload) => jobForm(row, reload) });
      if (can.del) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除任务 <b>${esc(row.name)}</b> 吗？`)) return;
        await api('/jobs', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });

  async function jobForm(row, reload) {
    const handlers = await api('/jobs/handlers');
    const opts = Object.entries(handlers).map(([k, v]) => ({ label: `${k}（${v}）`, value: k }));
    formModal({
      title: row ? '编辑任务：' + row.name : '新增任务',
      width: 'modal-lg',
      values: row || { status: 2, spec: '0 2 * * *' },
      fields: [
        { key: 'name', label: '任务名称', required: true },
        { key: 'handler', label: '处理器', type: 'select', options: opts, required: true },
        { key: 'spec', label: 'cron 表达式', required: true, placeholder: '0 2 * * *', help: '5 段式：分 时 日 月 周，如 0 2 * * * 表示每天凌晨 2 点' },
        { key: 'params', label: '参数（JSON）', type: 'textarea', rows: 2, placeholder: '{"days":90}' },
        { key: 'status', label: '状态', type: 'radio', options: [{ label: '启用', value: 1 }, { label: '停用', value: 2 }], required: true },
        { key: 'remark', label: '备注', type: 'textarea', rows: 2 },
      ],
      onSubmit: async v => {
        if (row) { await api('/jobs/' + row.id, { method: 'PUT', body: v }); toast('保存成功'); }
        else { await api('/jobs', { method: 'POST', body: v }); toast('创建成功'); }
        reload();
      },
    });
  }

  async function showJobLogs(row) {
    const res = await api('/job-logs', { query: { jobName: row.name, page: 1, size: 20 } });
    openModal({
      title: `任务日志：${row.name}`,
      width: 'modal-lg',
      body: renderTable([
        { title: '时间', render: r => `<span class="dim">${fmtTime(r.runAt)}</span>` },
        { title: '结果', render: r => r.success ? tag('成功', 'success') : tag('失败', 'danger') },
        { title: '耗时', render: r => `${r.costMs}ms` },
        { title: '输出/错误', render: r => `<span style="white-space:normal;display:inline-block;max-width:420px">${esc(r.output || r.error || '-')}</span>` },
        { title: '方式', render: r => r.manual ? tag('手动', 'info') : tag('调度', 'primary') },
      ], res.list || []),
      footer: '<button class="btn" onclick="this.closest(\'.modal-mask\').remove()">关 闭</button>',
    });
  }
};

// ============================ 通知公告 ============================
window.Pages.notices = function (container) {
  const can = {
    create: hasPerm('sys:notice:create'), update: hasPerm('sys:notice:update'), del: hasPerm('sys:notice:delete'),
  };
  listPage(container, {
    title: '通知公告', desc: '系统通知与公告发布',
    filters: [
      { key: 'title', label: '标题' },
      { key: 'type', label: '类型', type: 'select', options: [{ label: '通知', value: 1 }, { label: '公告', value: 2 }] },
      { key: 'status', label: '状态', type: 'select', options: [{ label: '已发布', value: 1 }, { label: '草稿', value: 2 }] },
    ],
    loadData: q => api('/notices', { query: q }),
    canAdd: can.create,
    onAdd: reload => noticeForm(null, reload),
    columns: [
      { title: '标题', render: r => `<b>${esc(r.title)}</b>` },
      { title: '类型', render: r => r.type === 2 ? tag('公告', 'warning') : tag('通知', 'info') },
      { title: '状态', render: r => statusTag(r.status, '已发布', '草稿') },
      { title: '发布人', key: 'publisher' },
      { title: '创建时间', render: r => `<span class="dim">${fmtTime(r.createdAt)}</span>` },
      { title: '操作', width: '160px' },
    ],
    rowActions: r => {
      const acts = [];
      if (can.update) acts.push({ label: '编辑', onClick: (row, reload) => noticeForm(row, reload) });
      if (can.del) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除公告 <b>${esc(row.title)}</b> 吗？`)) return;
        await api('/notices', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });

  function noticeForm(row, reload) {
    formModal({
      title: row ? '编辑公告：' + row.title : '新增公告',
      values: row || { type: 1, status: 1 },
      fields: [
        { key: 'title', label: '标题', required: true },
        { key: 'type', label: '类型', type: 'radio', options: [{ label: '通知', value: 1 }, { label: '公告', value: 2 }], required: true },
        { key: 'content', label: '内容', type: 'textarea', rows: 8, required: true },
        { key: 'status', label: '状态', type: 'radio', options: [{ label: '发布', value: 1 }, { label: '草稿', value: 2 }], required: true },
      ],
      onSubmit: async v => {
        if (row) { await api('/notices/' + row.id, { method: 'PUT', body: v }); toast('保存成功'); }
        else { await api('/notices', { method: 'POST', body: v }); toast('发布成功'); }
        reload();
      },
    });
  }
};

// ============================ 文件管理 ============================
window.Pages.files = function (container) {
  const can = { upload: hasPerm('sys:file:upload'), del: hasPerm('sys:file:delete') };
  listPage(container, {
    title: '文件管理', desc: '上传文件保存在服务端 uploads 目录',
    filters: [{ key: 'name', label: '文件名' }],
    loadData: q => api('/files', { query: q }),
    extraActions: (bar, reload) => {
      if (!can.upload) return;
      const input = htmlToEl('<input type="file" hidden>');
      const btn = htmlToEl('<button class="btn btn-primary">⬆ 上传文件</button>');
      bar.appendChild(input); bar.appendChild(btn);
      btn.onclick = () => input.click();
      input.onchange = async () => {
        if (!input.files.length) return;
        const fd = new FormData();
        fd.append('file', input.files[0]);
        btn.disabled = true; btn.textContent = '上传中...';
        try {
          const res = await fetch('/api/v1/files/upload', { method: 'POST', headers: { Authorization: 'Bearer ' + state.token }, body: fd });
          const data = await res.json();
          if (data.code !== 0) throw new Error(data.msg || '上传失败');
          toast('上传成功'); reload();
        } catch (e) { toastError(e); }
        finally { btn.disabled = false; btn.textContent = '⬆ 上传文件'; input.value = ''; }
      };
    },
    columns: [
      { title: '文件名', render: r => `<b>${esc(r.originalName)}</b>` },
      { title: '大小', render: r => fmtSize(r.size) },
      { title: '类型', render: r => `<span class="dim">${esc(r.contentType || '-')}</span>` },
      { title: 'URL', render: r => `<a href="${esc(r.url)}" target="_blank" class="mono">${esc(r.url)}</a>` },
      { title: '上传人', key: 'uploader' },
      { title: '时间', render: r => `<span class="dim">${fmtTime(r.createdAt)}</span>` },
      { title: '操作', width: '200px' },
    ],
    rowActions: r => {
      const acts = [{ label: '下载', onClick: () => window.open(`/api/v1/files/${r.id}/download?access_token=${encodeURIComponent(state.token)}`) }];
      if (can.del) acts.push({ label: '删除', danger: true, onClick: async (row, reload) => {
        if (!await confirmBox(`确定删除文件 <b>${esc(row.originalName)}</b> 吗？`)) return;
        await api('/files', { method: 'DELETE', body: { ids: [row.id] } });
        toast('删除成功'); reload();
      } });
      return acts;
    },
  });
};

// ============================ 系统监控 ============================
window.Pages.monitor = function (container) {
  container.innerHTML = `
    <div class="page-header">
      <div><div class="page-title">系统监控</div><div class="page-desc">主机与运行时状态（10 秒自动刷新）</div></div>
      <button class="btn" id="refresh-monitor">↻ 立即刷新</button>
    </div>
    <div id="monitor-body"><div class="card"><div class="empty" style="color:#94a3b8">加载中...</div></div></div>`;

  async function load() {
    let info;
    try { info = await api('/monitor/server'); } catch (e) { toastError(e); return; }
    const bar = (pct) => {
      const cls = pct > 85 ? 'danger' : pct > 60 ? 'warn' : '';
      return `<div class="progress"><div class="progress-bar ${cls}" style="width:${Math.min(pct, 100)}%"></div></div>`;
    };
    $('#monitor-body').innerHTML = `
      <div class="stat-grid" style="grid-template-columns:repeat(auto-fit,minmax(280px,1fr))">
        <div class="stat-card" style="display:block">
          <div class="mb8"><b>CPU</b> <span class="dim">${info.cpuCores} 核 · ${info.cpuPercent}%</span></div>
          ${bar(info.cpuPercent)}
        </div>
        <div class="stat-card" style="display:block">
          <div class="mb8"><b>内存</b> <span class="dim">${fmtSize(info.memUsed)} / ${fmtSize(info.memTotal)} · ${info.memPercent}%</span></div>
          ${bar(info.memPercent)}
        </div>
        <div class="stat-card" style="display:block">
          <div class="mb8"><b>磁盘（${esc(info.diskPath)}）</b> <span class="dim">${fmtSize(info.diskTotal - info.diskFree)} / ${fmtSize(info.diskTotal)} · ${info.diskPercent}%</span></div>
          ${bar(info.diskPercent)}
        </div>
      </div>
      <div class="profile-grid mt16">
        <div class="card"><div class="card-title">主机信息</div>
          <table class="table">
            <tr><th style="width:120px">主机名</th><td>${esc(info.hostname)}</td></tr>
            <tr><th>操作系统</th><td>${esc(info.os)}（${esc(info.platform)}）</td></tr>
            <tr><th>内核架构</th><td>${esc(info.kernelArch)}</td></tr>
            <tr><th>开机时长</th><td>${fmtUptime(info.hostUptime)}</td></tr>
          </table>
        </div>
        <div class="card"><div class="card-title">运行时信息</div>
          <table class="table">
            <tr><th style="width:120px">Go 版本</th><td>${esc(info.goVersion)}</td></tr>
            <tr><th>Gin 版本</th><td>${esc(info.ginVersion)}</td></tr>
            <tr><th>Goroutines</th><td>${info.numGoroutine}</td></tr>
            <tr><th>堆内存</th><td>${info.heapAllocMB} MB</td></tr>
            <tr><th>应用运行时长</th><td>${fmtUptime(info.appUptimeS)}</td></tr>
            <tr><th>采集时间</th><td class="dim">${esc(info.now)}</td></tr>
          </table>
        </div>
      </div>`;
  }

  $('#refresh-monitor').onclick = load;
  load();
  registerTimer(setInterval(load, 10000));
};

// ============================ 个人中心 ============================
window.Pages.profile = async function (container) {
  const profile = await api('/auth/profile');
  const u = profile.user;
  const genders = await dict('sys_gender');

  container.innerHTML = `
    <div class="page-header"><div><div class="page-title">个人中心</div><div class="page-desc">维护个人资料与登录密码</div></div></div>
    <div class="profile-grid">
      <div class="card">
        <div class="card-title">个人资料</div>
        <form id="profile-form">
          <div class="form-item"><label>用户名</label><input type="text" value="${esc(u.username)}" disabled></div>
          <div class="form-item"><label>昵称 <span class="req">*</span></label><input type="text" name="nickname" value="${esc(u.nickname)}" required></div>
          <div class="form-item"><label>邮箱</label><input type="email" name="email" value="${esc(u.email || '')}"></div>
          <div class="form-item"><label>手机号</label><input type="text" name="phone" value="${esc(u.phone || '')}"></div>
          <div class="form-item"><label>性别</label>
            <select name="gender">${genders.map(g => `<option value="${esc(g.value)}" ${String(u.gender || 0) === String(g.value) ? 'selected' : ''}>${esc(g.label)}</option>`).join('')}</select>
          </div>
          <button class="btn btn-primary" type="submit">保存资料</button>
        </form>
      </div>
      <div class="card">
        <div class="card-title">修改密码</div>
        <form id="pwd-form">
          <div class="form-item"><label>原密码 <span class="req">*</span></label><input type="password" name="oldPassword" required></div>
          <div class="form-item"><label>新密码 <span class="req">*</span></label><input type="password" name="newPassword" required minlength="6" placeholder="至少 6 位"></div>
          <div class="form-item"><label>确认新密码 <span class="req">*</span></label><input type="password" name="confirm" required></div>
          <button class="btn btn-primary" type="submit">修改密码</button>
        </form>
        <div class="card-title mt16">我的角色</div>
        ${(profile.roles || []).map(r => tag(r, 'primary')).join(' ') || '<span class="dim">未分配角色</span>'}
      </div>
    </div>`;

  $('#profile-form').addEventListener('submit', async e => {
    e.preventDefault();
    const f = e.target;
    try {
      await api('/auth/profile', {
        method: 'PUT',
        body: { nickname: f.nickname.value.trim(), email: f.email.value.trim(), phone: f.phone.value.trim(), gender: Number(f.gender.value) },
      });
      toast('资料已保存');
      state.user.nickname = f.nickname.value.trim();
      $('#user-nickname').textContent = state.user.nickname;
    } catch (err) { toastError(err); }
  });

  $('#pwd-form').addEventListener('submit', async e => {
    e.preventDefault();
    const f = e.target;
    if (f.newPassword.value !== f.confirm.value) { toast('两次输入的新密码不一致', 'error'); return; }
    try {
      await api('/auth/profile/password', { method: 'PUT', body: { oldPassword: f.oldPassword.value, newPassword: f.newPassword.value } });
      toast('密码修改成功，下次登录请使用新密码');
      f.reset();
    } catch (err) { toastError(err); }
  });
};
