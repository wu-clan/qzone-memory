// QQ 空间回忆 - 前端应用
const app = {
  qq: "",
  nickname: "",
  currentView: "memories",
  currentMemoryFilter: "all",
  searchKeyword: "",
  interactionKeyword: "",
  searchTimer: null,
  unifiedSearchRequestId: 0,
  loginMode: "qr",
  pollTimer: null,
  syncTimer: null,
  qrRequestId: 0,
  syncStarting: false,
  page: 1,
  pageSize: 20,
  loading: false,
  albumDetailOpen: false,
  albumDetailId: "",
  albumDetailName: "",
  albumPhotoPage: 1,
  lightboxImages: [],
  lightboxIndex: 0,
  infiniteObserver: null,
  deferredImageObserver: null,
  friendSearch: "",
  friendFilter: "all",
  friendGroupFilter: "all",
  lastTimelineGroup: "",
  imagePlaceholder:
    "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==",
  viewMeta: {
    memories: {
      kicker: "回忆时间线",
      title: "完整档案",
      description: "按时间回看 QQ 空间的说说、相册、留言、互动与早已删除的痕迹。",
      emptyTitle: "档案还是空的",
      emptyDescription: "还没有可展示的数据，先同步一次，让说说、相册、留言和日志重新归档。",
    },
    friends: {
      kicker: "旧友名册",
      title: "好友与旧联系人",
      description: "从联系人重新进入共同的回忆——当前好友、分组、特别关心，以及由历史互动留下的旧联系人。",
      emptyTitle: "名册还是空的",
      emptyDescription: "还没有可展示的好友或历史联系人，先同步一次联系人与互动记录。",
    },
    interactions: {
      kicker: "好友互动",
      title: "找一个人，看全部往来",
      description: "输入昵称或 QQ，聚合 TA 的点赞、评论、留言、访客与内容提及。",
      emptyTitle: "先输入一个好友",
      emptyDescription: "在右上角搜索框输入昵称或 QQ，查看你和 TA 的全部互动。",
    },
    onthisday: {
      kicker: "那年今日",
      title: "历史上的今天",
      description: "在过去的同月同日，你曾留下这些痕迹。",
      emptyTitle: "今天还没有回忆",
      emptyDescription: "历史上的今天暂时没有可展示的内容，换一天，或先同步更多数据。",
    },
  },

  // ===== 初始化 =====
  init() {
    this.qq = localStorage.getItem("qzone_qq") || "";
    this.nickname = localStorage.getItem("qzone_nickname") || "";
    if (this.qq) {
      this.checkLogin();
    } else {
      this.showLogin();
    }
  },

  async checkLogin() {
    try {
      const res = await this.api(`/api/v1/login/user?qq=${this.qq}`);
      if (res.code === 0 && res.data) {
        this.nickname = res.data.nickname || this.qq;
        localStorage.setItem("qzone_nickname", this.nickname);
        this.showMain();
      } else {
        this.showLogin();
      }
    } catch {
      this.showLogin();
    }
  },

  // ===== 页面切换 =====
  showLogin() {
    this.qq = "";
    localStorage.removeItem("qzone_qq");
    localStorage.removeItem("qzone_nickname");
    document.getElementById("login-page").classList.add("active");
    document.getElementById("main-page").classList.remove("active");
    this.resetLoginForm();
    this.switchLoginMode("qr");
  },

  showMain() {
    document.getElementById("login-page").classList.remove("active");
    document.getElementById("main-page").classList.add("active");
    this.stopPoll();

    // 设置用户信息
    const name = this.cleanName(this.nickname, "我的空间");
    const avatar = document.getElementById("user-avatar");
    avatar.src = this.proxyImageUrl(`https://q.qlogo.cn/headimg_dl?dst_uin=${this.qq}&spec=100`);
    avatar.alt = `${name} 的头像`;
    avatar.onerror = () => {
      avatar.onerror = null;
      avatar.classList.add("is-failed");
    };
    document.getElementById("user-name").textContent = name;
    const qqEl = document.getElementById("user-qq");
    if (qqEl) qqEl.textContent = "QQ " + this.qq;

    this.renderProfile();
    this.renderRailWidgets();
    this.loadData();
  },

  // 右栏挂件：那年今日 + 最近访客（真头像）
  async renderRailWidgets() {
    const dateEl = document.getElementById("onthisday-date");
    const now = new Date();
    if (dateEl) dateEl.textContent = `${now.getMonth() + 1} 月 ${now.getDate()} 日`;
    try {
      const r = await this.api(`/api/v1/memory/on-this-day?qq=${this.qq}`);
      const body = document.getElementById("onthisday-body");
      if (body && r.code === 0 && r.data) {
        const n = r.data.total || 0;
        body.textContent = n > 0 ? `历史上的今天，你留下过 ${n} 条回忆。` : "历史上的今天暂无记录。";
      }
    } catch {}
    try {
      const v = await this.api(`/api/v1/visitors?qq=${this.qq}&page=1&page_size=8`);
      const list = document.getElementById("visitor-list");
      const tot = document.getElementById("visitor-total");
      if (v.code === 0 && v.data) {
        if (tot) tot.textContent = (v.data.total || 0) + " 位";
        const items = Array.isArray(v.data.list) ? v.data.list : [];
        if (list) {
          list.innerHTML = items.slice(0, 8).map((it) => {
            const qq = it.visitor_qq || "";
            const name = this.cleanName(it.visitor_name, qq || "访客");
            const av = this.proxyImageUrl(it.avatar || `https://q.qlogo.cn/headimg_dl?dst_uin=${qq}&spec=100`);
            return `<div class="vz"><img src="${this.escapeHtml(av)}" onerror="this.style.visibility='hidden'"><span>${this.escapeHtml(name)}</span></div>`;
          }).join("");
        }
      }
    } catch {}
  },

  // 渲染个人中心头部（头像 + 昵称 + 说说/获赞/好友/年）
  async renderProfile() {
    const av = document.getElementById("pf-avatar");
    if (av) {
      av.src = this.proxyImageUrl(`https://q.qlogo.cn/headimg_dl?dst_uin=${this.qq}&spec=100`);
      av.onerror = () => { av.onerror = null; av.removeAttribute("src"); };
    }
    const nameEl = document.getElementById("pf-name");
    if (nameEl) nameEl.textContent = this.cleanName(this.nickname, "我的空间");
    const subEl = document.getElementById("pf-sub");
    if (subEl) subEl.textContent = "QQ " + this.qq + " · 这里收着你的全部回忆";

    const set = (id, v) => {
      const el = document.getElementById(id);
      if (el) el.textContent = this.formatCount(v || 0);
    };
    try {
      const res = await this.api(`/api/v1/memory/report?qq=${this.qq}`);
      if (res.code === 0 && res.data) {
        const by = res.data.by_type || {};
        set("pf-talks", by.talk || 0);
        set("pf-likes", by.like || 0);
        const y = document.getElementById("pf-years");
        if (y) y.textContent = res.data.span_years || 0;
      }
    } catch {}
    try {
      const fr = await this.api(`/api/v1/friends?qq=${this.qq}&page=1&page_size=1`);
      if (fr.code === 0 && fr.data) set("pf-friends", fr.data.total || 0);
    } catch {}
  },

  logout() {
    this.stopPoll();
    this.stopSyncPoll();
    this.showLogin();
  },

  // ===== 二维码登录 =====
  switchLoginMode(mode) {
    this.loginMode = mode === "input" ? "input" : "qr";

    const qrBtn = document.getElementById("login-mode-qr");
    const inputBtn = document.getElementById("login-mode-input");
    const qrPanel = document.getElementById("login-panel-qr");
    const inputPanel = document.getElementById("login-panel-input");

    qrBtn.classList.toggle("active", this.loginMode === "qr");
    inputBtn.classList.toggle("active", this.loginMode === "input");
    qrPanel.classList.toggle("hidden", this.loginMode !== "qr");
    qrPanel.classList.toggle("active", this.loginMode === "qr");
    inputPanel.classList.toggle("hidden", this.loginMode !== "input");
    inputPanel.classList.toggle("active", this.loginMode === "input");

    if (this.loginMode === "qr") {
      this.loadQRCode();
      return;
    }

    this.qrRequestId += 1;
    this.stopPoll();
  },

  async loadQRCode() {
    this.stopPoll();
    const requestId = ++this.qrRequestId;
    const loading = document.getElementById("qr-loading");
    const img = document.getElementById("qr-image");
    const expired = document.getElementById("qr-expired");

    loading.classList.remove("hidden");
    img.classList.add("hidden");
    img.src = "";
    expired.classList.add("hidden");
    this.setStatus("正在生成二维码", "");

    try {
      const res = await this.api("/api/v1/login/qrcode");
      if (this.loginMode !== "qr" || requestId !== this.qrRequestId) return;
      if (res.code === 0 && res.data.qr_image) {
        loading.classList.add("hidden");
        img.src = res.data.qr_image;
        img.classList.remove("hidden");
        this.setStatus("请使用手机 QQ 扫码", "");
        this.startPoll();
        return;
      }
      loading.classList.add("hidden");
      this.setStatus(res.message || "请求失败", "expired");
    } catch (e) {
      if (requestId !== this.qrRequestId) return;
      loading.classList.add("hidden");
      this.setStatus(e.message || "请求失败", "expired");
    }
  },

  refreshQRCode() {
    this.stopPoll();
    this.loadQRCode();
  },

  startPoll() {
    this.stopPoll();
    this.pollTimer = setInterval(() => this.pollStatus(), 2000);
  },

  stopPoll() {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
  },

  async pollStatus() {
    try {
      const res = await this.api("/api/v1/login/status");
      if (res.code !== 0) {
        if (res.code === 410) {
          this.stopPoll();
          document.getElementById("qr-expired").classList.remove("hidden");
          this.setStatus(res.message || "二维码已失效，请重新获取", "expired");
          return;
        }

        this.setStatus(res.message || "正在等待二维码状态", "");
        return;
      }

      const data = res.data;
      switch (data.status) {
        case 0:
          this.setStatus("二维码已生成，等待扫码", "");
          break;
        case 1:
          this.setStatus(
            `${data.nickname || "当前账号"}已扫码，请在手机 QQ 中确认`,
            "scanned",
          );
          break;
        case 2:
          this.stopPoll();
          this.setStatus("登录成功，正在进入", "success");
          await this.handleLoginSuccess(data);
          break;
        case 3:
          this.stopPoll();
          document.getElementById("qr-expired").classList.remove("hidden");
          this.setStatus("二维码已失效，请重新获取", "expired");
          break;
        case 4:
          this.stopPoll();
          document.getElementById("qr-expired").classList.remove("hidden");
          this.setStatus("本次登录已取消，请重新获取二维码", "expired");
          break;
      }
    } catch {
      this.setStatus("正在等待二维码状态", "");
    }
  },

  async fetchUserAfterLogin() {
    // 登录成功后，当前流程已直接从轮询结果中取得 QQ 号和昵称
  },

  async handleLoginSuccess(data) {
    this.qq = data.qq || "";
    this.nickname = data.nickname || "";
    if (this.qq) {
      localStorage.setItem("qzone_qq", this.qq);
      localStorage.setItem("qzone_nickname", this.nickname);
      setTimeout(() => this.showMain(), 300);
    }
  },

  setStatus(text, type) {
    document.getElementById("status-text").textContent = text;
    const dot = document.querySelector(".status-dot");
    dot.className = "status-dot";
    if (type) dot.classList.add(type);
  },

  resetLoginForm() {
    const input = document.getElementById("qq-input");
    if (input) input.value = "";
    this.setInputStatus("");
  },

  setInputStatus(text, type = "") {
    const status = document.getElementById("login-input-status");
    if (!status) return;

    if (!text) {
      status.textContent = "";
      status.className = "form-hint hidden";
      return;
    }

    status.textContent = text;
    status.className = "form-hint";
    if (type) status.classList.add(type);
  },

  async submitQQLogin(event) {
    event.preventDefault();
    const input = document.getElementById("qq-input");
    const qq = (input.value || "").trim();
    this.setInputStatus("");

    try {
      const res = await this.api(`/api/v1/login/user?qq=${qq}`);
      if (res.code === 0 && res.data) {
        this.qq = qq;
        this.nickname = res.data.nickname || qq;
        localStorage.setItem("qzone_qq", this.qq);
        localStorage.setItem("qzone_nickname", this.nickname);
        setTimeout(() => this.showMain(), 200);
        return;
      }
      this.setInputStatus(res.message || "请求失败", "error");
    } catch {
      this.setInputStatus("请求失败", "error");
    }
  },

  // ===== 数据加载 =====
  async loadData(options = {}) {
    const resetScroll = options.resetScroll !== false;
    this.page = 1;
    this.closeAlbumDetail();
    const timeline = document.getElementById("timeline");
    const friendsView = document.getElementById("friends-view");
    const empty = document.getElementById("empty-state");
    const loadingState = document.getElementById("loading-state");
    const contentScroll = document.getElementById("content-scroll");
    const previousScrollTop = contentScroll ? contentScroll.scrollTop : 0;

    if (resetScroll && contentScroll) contentScroll.scrollTop = 0;

    if (resetScroll) {
      timeline.classList.add("hidden");
      friendsView.classList.add("hidden");
      empty.classList.add("hidden");
      this.setLoadMoreState("hidden");
    }

    const isMemories = this.currentView === "memories";
    const isFriends = this.currentView === "friends";
    const isInteractions = this.currentView === "interactions";
    document.getElementById("memory-filters").classList.toggle("hidden", !isMemories);
    const memorySearchField = document.getElementById("memory-search-field");
    if (memorySearchField) memorySearchField.classList.toggle("hidden", !(isMemories || isInteractions));
    document.getElementById("friend-toolbar").classList.toggle("hidden", !isFriends);

    this.updateUnifiedSearchField();
    this.updateContentHead();
    this.updateRailHeader();
    this.updateSyncStatus("检查中", "正在确认是否存在进行中的同步任务", "running");
    this.setSyncButtonState(false);

    // 先检查同步状态
    const progressRes = await this.api("/api/v1/sync/progress");
    if (progressRes.code === 0 && progressRes.data) {
      this.applySyncStatus(progressRes.data);
      if (progressRes.data.status === "running") {
        this.showSyncProgress();
        return;
      }
    } else {
      this.updateSyncStatus("状态未知", "暂时无法获取同步任务状态", "");
    }

    if (loadingState && resetScroll) loadingState.classList.remove("hidden");
    const payload = await this.fetchCurrentViewData();
    if (loadingState) loadingState.classList.add("hidden");

    const items = Array.isArray(payload) ? payload : payload?.list || [];

    if (!items || items.length === 0) {
      timeline.classList.add("hidden");
      friendsView.classList.add("hidden");
      this.setLoadMoreState("hidden");
      empty.classList.remove("hidden");
      this.updateEmptyState();
    } else if (this.currentView === "friends") {
      this.renderFriends(payload);
      timeline.classList.add("hidden");
      empty.classList.add("hidden");
      this.setLoadMoreState("hidden");
      friendsView.classList.remove("hidden");
      this.renderFriendRail(payload);
    } else {
      this.renderTimeline(items);
      empty.classList.add("hidden");
      friendsView.classList.add("hidden");
      timeline.classList.remove("hidden");
      if (["memories", "interactions"].includes(this.currentView) && items.length >= this.pageSize) {
        this.setLoadMoreState("idle");
      } else {
        this.setLoadMoreState("hidden");
      }
      this.setupInfiniteLoad();
    }

    if (!resetScroll && contentScroll) {
      this.restoreScrollTop(contentScroll, previousScrollTop);
    }

    this.loadCounts();
    if (this.currentView === "memories") {
      this.loadMemoryStats();
    }
  },

  async fetchCurrentViewData() {
    const qq = this.qq;
    const p = this.page;
    const ps = this.pageSize;

    let url = "";
    if (this.currentView === "friends") {
      return this.fetchAllFriendsData();
    } else if (this.currentView === "interactions") {
      if (!this.interactionKeyword) return { list: [], total: 0 };
      url = `/api/v1/memory/interactions?qq=${qq}&keyword=${encodeURIComponent(this.interactionKeyword)}&page=${p}&page_size=${ps}`;
    } else if (this.currentView === "onthisday") {
      url = `/api/v1/memory/on-this-day?qq=${qq}`;
    } else if (this.searchKeyword) {
      url = `/api/v1/memory/search?qq=${qq}&keyword=${encodeURIComponent(this.searchKeyword)}&page=${p}&page_size=${ps}`;
    } else {
      url = `/api/v1/memory/timeline?qq=${qq}&type=${encodeURIComponent(this.currentMemoryFilter)}&page=${p}&page_size=${ps}`;
    }

    try {
      const res = await this.api(url);
      if (res.code !== 0 || !res.data) return this.currentView === "friends" ? null : [];
      if (this.currentView === "friends") return res.data;
      if (this.currentView === "interactions") {
        this.renderInteractionMeta(res.data);
        return res.data.items || { list: [], total: 0 };
      }
      return res.data.list || [];
    } catch {}
    return this.currentView === "friends" ? null : [];
  },

  async fetchAllFriendsData() {
    const qq = this.qq;
    const pageSize = 100;
    let page = 1;
    let total = 0;
    let currentTotal = 0;
    let groupTotal = 0;
    let historicalTotal = 0;
    let groups = [];
    const list = [];

    try {
      while (true) {
        const res = await this.api(
          `/api/v1/friends?qq=${qq}&include_deleted=true&page=${page}&page_size=${pageSize}`,
        );
        if (res.code !== 0 || !res.data) break;

        const data = res.data;
        const pageList = Array.isArray(data.list) ? data.list : [];
        if (page === 1) {
          total = data.total || 0;
          currentTotal = data.current_total || 0;
          historicalTotal = data.historical_total || 0;
          groupTotal = data.group_total || 0;
          groups = Array.isArray(data.groups) ? data.groups : [];
        }

        list.push(...pageList);

        if (pageList.length < pageSize || list.length >= total) {
          break;
        }
        page += 1;
      }
    } catch {}

    return {
      list,
      groups,
      total,
      current_total: currentTotal,
      historical_total: historicalTotal,
      group_total: groupTotal,
      page: 1,
      page_size: list.length || pageSize,
    };
  },

  setLoadMoreState(state = "idle") {
    const loadMore = document.getElementById("load-more");
    if (!loadMore) return;
    const button = loadMore.querySelector("button");
    loadMore.classList.toggle("hidden", state === "hidden");
    loadMore.classList.toggle("is-loading", state === "loading");
    if (button) {
      button.disabled = state === "loading";
      button.textContent = state === "loading" ? "正在加载…" : "加载更多";
    }
  },

  restoreScrollTop(scroller, top) {
    const restore = () => {
      const maxTop = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
      scroller.scrollTop = Math.min(Math.max(top, 0), maxTop);
    };
    restore();
    requestAnimationFrame(restore);
  },

  async loadMore() {
    if (this.loading || !["memories", "interactions"].includes(this.currentView)) return;
    this.loading = true;
    if (this.infiniteObserver) this.infiniteObserver.disconnect();
    this.setLoadMoreState("loading");
    const previousPage = this.page;
    this.page++;

    try {
      const payload = await this.fetchCurrentViewData();
      const items = Array.isArray(payload) ? payload : payload?.list || [];
      if (!items.length) {
        this.setLoadMoreState("hidden");
        return;
      }
      this.appendTimeline(items);
      if (items.length >= this.pageSize) {
        this.setLoadMoreState("idle");
      } else {
        this.setLoadMoreState("hidden");
      }
    } catch {
      this.page = previousPage;
      this.setLoadMoreState("idle");
    } finally {
      this.loading = false;
      this.setupInfiniteLoad();
    }
  },

  async loadCounts() {
    let memoryCount = 0;
    let friendCount = 0;
    try {
      const memoryRes = await this.api(`/api/v1/memory/timeline?qq=${this.qq}&page=1&page_size=1`);
      memoryCount = memoryRes.code === 0 && memoryRes.data ? memoryRes.data.total || 0 : 0;
    } catch {}
    try {
      const friendRes = await this.api(`/api/v1/friends?qq=${this.qq}&page=1&page_size=1`);
      friendCount = friendRes.code === 0 && friendRes.data ? friendRes.data.total || 0 : 0;
    } catch {}

    this.setNavCount("nav-count-memories", memoryCount);
    this.setNavCount("nav-count-friends", friendCount);
  },

  setNavCount(id, value) {
    const el = document.getElementById(id);
    if (!el) return;
    el.textContent = value > 0 ? this.formatCount(value) : "—";
  },

  async loadMemoryStats() {
    try {
      const res = await this.api(`/api/v1/memory/stats?qq=${this.qq}`);
      if (res.code !== 0 || !res.data) return;
      if (this.currentView === "memories") this.renderMemoryRail(res.data);
    } catch {}
  },

  // ===== 右侧档案概览 =====
  updateRailHeader() {
    const kicker = document.getElementById("rail-kicker");
    const totalLabel = document.getElementById("rail-total-label");
    const breakdownTitle = document.getElementById("rail-breakdown-title");
    const yearsCard = document.getElementById("rail-years-card");
    const isFriends = this.currentView === "friends";
    const isInteractions = this.currentView === "interactions";
    if (kicker) kicker.textContent = isFriends ? "名册概览" : isInteractions ? "互动概览" : "档案概览";
    if (totalLabel) totalLabel.textContent = isFriends ? "位联系人" : isInteractions ? "条互动" : "条回忆";
    if (breakdownTitle) breakdownTitle.textContent = isFriends ? "名册构成" : isInteractions ? "互动构成" : "内容构成";
    if (yearsCard) yearsCard.classList.toggle("hidden", isFriends || isInteractions);
  },

  renderRailBars(containerId, rows, max) {
    const el = document.getElementById(containerId);
    if (!el) return;
    const ceiling = Math.max(max || 0, 1);
    el.innerHTML = rows
      .map(([name, count, cls]) => {
        const pct = count > 0 ? Math.max(4, Math.round((count / ceiling) * 100)) : 0;
        return `<div class="rail-stat-row ${cls || ""}">
          <div class="rail-stat-top">
            <span class="rail-stat-name">${this.escapeHtml(name)}</span>
            <span class="rail-stat-value">${this.formatCount(count)}</span>
          </div>
          <div class="rail-bar"><i style="width:${pct}%"></i></div>
        </div>`;
      })
      .join("");
  },

  renderMemoryRail(stats) {
    const total = stats.total || 0;
    const byType = stats.by_type || {};
    const byYear = Array.isArray(stats.by_year) ? stats.by_year : [];

    const totalEl = document.getElementById("rail-total");
    if (totalEl) totalEl.textContent = this.formatCount(total);

    const labels = {
      activity: "动态", talk: "说说", blog: "日志", album: "相册",
      message: "留言", comment: "评论", visitor: "访客", video: "视频",
      like: "点赞", favorite: "收藏", diary: "日记", mention: "提及", share: "转发",
    };
    const entries = Object.entries(byType)
      .filter(([, v]) => v > 0)
      .sort((a, b) => b[1] - a[1]);

    if (entries.length) {
      const rows = entries.map(([type, count]) => [
        labels[type] || type,
        count,
        `type-${type}`,
      ]);
      this.renderRailBars("rail-stats", rows, entries[0][1]);
    }

    const yearsEl = document.getElementById("rail-years");
    if (yearsEl && byYear.length) {
      const maxYear = byYear.reduce((m, y) => Math.max(m, y.count || 0), 1);
      yearsEl.innerHTML = byYear
        .map((y) => {
          const pct = Math.max(3, Math.round(((y.count || 0) / maxYear) * 100));
          return `<div class="rail-year-row">
            <span class="rail-year-label">${this.escapeHtml(String(y.year))}</span>
            <div class="rail-year-bar"><i style="width:${pct}%"></i></div>
            <span class="rail-year-count">${this.formatCount(y.count || 0)}</span>
          </div>`;
        })
        .join("");
    }
  },

  renderFriendRail(data) {
    const total = data.total || (data.list ? data.list.length : 0);
    const special = (data.list || []).filter((f) => f && f.is_special_care).length;

    const totalEl = document.getElementById("rail-total");
    if (totalEl) totalEl.textContent = this.formatCount(total);

    const rows = [
      ["当前好友", data.current_total || 0, "type-talk"],
      ["历史联系人", data.historical_total || 0, "type-diary"],
      ["好友分组", data.group_total || 0, "type-activity"],
      ["特别关心", special, "type-like"],
    ];
    const max = rows.reduce((m, r) => Math.max(m, r[1]), 1);
    this.renderRailBars("rail-stats", rows, max);
  },

  switchView(view, event) {
    if (event) event.preventDefault();
    this.currentView = view;
    this.clearMemorySearch();
    this.setActiveNav(view);
    this.page = 1;
    this.loadData();
    if (view === "interactions") {
      setTimeout(() => document.getElementById("memory-search")?.focus(), 50);
    }
  },

  setActiveNav(view = this.currentView) {
    const navView = view === "interactions" ? "memories" : view;
    document
      .querySelectorAll(".nav-item")
      .forEach((el) => el.classList.toggle("active", el.dataset.view === navView));
  },

  switchMemoryFilter(filter, event) {
    if (event) event.preventDefault();
    this.currentMemoryFilter = filter;
    this.clearMemorySearch();
    document
      .querySelectorAll("#memory-filters .chip")
      .forEach((el) => el.classList.toggle("active", el.dataset.filter === filter));
    this.loadData();
  },

  handleMemorySearch(event) {
    const value = (event.target.value || "").trim();
    if (this.searchTimer) clearTimeout(this.searchTimer);
    this.searchTimer = setTimeout(() => {
      this.runUnifiedSearch(value);
    }, 300);
  },

  clearMemorySearch() {
    this.unifiedSearchRequestId++;
    this.searchKeyword = "";
    this.interactionKeyword = "";
    if (this.searchTimer) {
      clearTimeout(this.searchTimer);
      this.searchTimer = null;
    }
    const input = document.getElementById("memory-search");
    if (input) input.value = "";
  },

  handleUnifiedSearchKey(event) {
    if (event.key === "Enter") {
      event.preventDefault();
      if (this.searchTimer) {
        clearTimeout(this.searchTimer);
        this.searchTimer = null;
      }
      const input = document.getElementById("memory-search");
      this.runUnifiedSearch((input?.value || "").trim(), { forceProbe: true });
    }
  },

  async runUnifiedSearch(keyword, options = {}) {
    const value = (keyword || "").trim();
    const requestId = ++this.unifiedSearchRequestId;

    if (!value) {
      this.currentView = "memories";
      this.searchKeyword = "";
      this.interactionKeyword = "";
      this.page = 1;
      this.setActiveNav("memories");
      this.loadData();
      return;
    }

    let useInteractions = this.isQQSearch(value);
    if (!useInteractions && this.shouldProbeFriendSearch(value, options.forceProbe)) {
      const data = await this.fetchFriendSearchProbe(value);
      if (requestId !== this.unifiedSearchRequestId) return;
      useInteractions = (data.friend_candidates || []).length > 0;
    }

    this.currentView = useInteractions ? "interactions" : "memories";
    this.searchKeyword = useInteractions ? "" : value;
    this.interactionKeyword = useInteractions ? value : "";
    this.page = 1;
    this.setActiveNav(this.currentView);
    this.loadData();
  },

  isQQSearch(keyword) {
    return /^\d{5,12}$/.test((keyword || "").trim());
  },

  shouldProbeFriendSearch(keyword, forceProbe = false) {
    const value = (keyword || "").trim();
    if (!value) return false;
    if (forceProbe) return true;
    return Array.from(value).length >= 2;
  },

  async fetchFriendSearchProbe(keyword) {
    try {
      const res = await this.api(
        `/api/v1/memory/interactions?qq=${this.qq}&keyword=${encodeURIComponent(keyword)}&page=1&page_size=1&candidates_only=true`,
      );
      if (res.code === 0 && res.data) return res.data;
    } catch {}
    return { friend_candidates: [] };
  },

  searchInteractions() {
    const input = document.getElementById("memory-search");
    const keyword = (input?.value || "").trim();
    this.searchKeyword = "";
    this.interactionKeyword = keyword;
    this.currentView = "interactions";
    this.setActiveNav("interactions");
    this.page = 1;
    this.loadData();
  },

  searchInteractionsFor(keyword) {
    const input = document.getElementById("memory-search");
    if (input) input.value = keyword || "";
    this.interactionKeyword = keyword || "";
    this.searchInteractions();
  },

  renderInteractionMeta(data = {}) {
    const total = data.items?.total || 0;
    const first = (data.friend_candidates || [])[0];

    const totalEl = document.getElementById("rail-total");
    if (totalEl) totalEl.textContent = this.formatCount(total);
    const title = document.getElementById("rail-breakdown-title");
    if (title) title.textContent = "互动构成";
    const stats = data.stats || {};
    const rows = Object.entries(stats)
      .sort((a, b) => b[1] - a[1])
      .map(([type, count]) => [this.typeLabels[type] || type, count, `type-${type}`]);
    this.renderRailBars("rail-stats", rows, rows[0]?.[1] || 1);

    if (first) {
      const name = this.cleanName(first.name || first.remark, first.qq);
      const status = first.is_deleted || first.is_current === false ? "历史联系人" : "当前好友";
      const group = first.group_name ? ` · ${first.group_name}` : "";
      const title = document.getElementById("content-title");
      const desc = document.getElementById("content-description");
      if (title) title.textContent = `与 “${name}” 的互动`;
      if (desc) desc.textContent = `QQ ${first.qq} · ${status}${group} · 找到 ${this.formatCount(total)} 条点赞、评论、留言、访客与内容提及`;
    }
  },

  // ===== 时间线渲染 =====
  typeLabels: {
    activity: "动态", talk: "说说", blog: "日志", album: "相册",
    message: "留言", comment: "评论", visitor: "访客", video: "视频",
    like: "点赞", favorite: "收藏", diary: "日记", photo: "照片",
    mention: "提及", share: "转发", other: "动态",
  },

  itemTime(item) {
    return (
      item.publish_time ||
      item.message_time ||
      item.mention_time ||
      item.create_time ||
      item.created_at ||
      ""
    );
  },

  renderTimeline(items) {
    const container = document.getElementById("timeline");
    container.innerHTML = "";
    let currentKey = "";
    let section = null;
    items.forEach((item) => {
      const key = this.formatTimelineGroup(this.itemTime(item));
      if (key !== currentKey || !section) {
        currentKey = key;
        section = this.createTimelineSection(key);
        container.appendChild(section);
      }
      this.addItemToSection(section, item);
    });
    this.lastTimelineGroup = currentKey;
    this.observeDeferredImages(container);
  },

  appendTimeline(items) {
    const container = document.getElementById("timeline");
    let lastKey = this.lastTimelineGroup || "";
    let section = container.lastElementChild;
    items.forEach((item) => {
      const key = this.formatTimelineGroup(this.itemTime(item));
      if (key !== lastKey || !section) {
        lastKey = key;
        section = this.createTimelineSection(key);
        container.appendChild(section);
      }
      this.addItemToSection(section, item);
    });
    this.lastTimelineGroup = lastKey;
    this.observeDeferredImages(container);
  },

  createTimelineSection(key) {
    const div = document.createElement("div");
    div.className = "timeline-section";
    div.dataset.group = key;
    let railHTML;
    if (key === "earlier") {
      railHTML = `<span class="timeline-rail-year">更早</span>
        <span class="timeline-rail-month">以前</span>
        <span class="timeline-rail-count" data-count>0 条</span>`;
    } else {
      const [year, month] = key.split("-");
      railHTML = `<span class="timeline-rail-year">${this.escapeHtml(year)}</span>
        <span class="timeline-rail-month">${this.escapeHtml(month)} 月</span>
        <span class="timeline-rail-count" data-count>0 条</span>`;
    }
    div.innerHTML = `<div class="timeline-rail">${railHTML}</div>
      <div class="timeline-items"></div>`;
    return div;
  },

  addItemToSection(section, item) {
    const items = section.querySelector(".timeline-items");
    items.appendChild(this.createTimelineItem(item));
    const counter = section.querySelector("[data-count]");
    if (counter) counter.textContent = `${items.childElementCount} 条`;
  },

  createTimelineItem(item) {
    const type = this.detectType(item);
    const div = document.createElement("article");
    div.className = `timeline-item post type-${type}` + (item.is_deleted ? " deleted" : "");

    const time = this.formatTime(this.itemTime(item));
    const author = this.displayAuthor(item);
    const authorQQ = item.author_qq || item.user_qq || this.qq;
    const title = String(item.title || "").trim();
    const content = String(item.content || item.summary || item.name || "").trim();

    let textHTML = "";
    if (type === "album") {
      textHTML = title ? `<div class="post-text">${this.escapeHtml(title)}</div>` : "";
    } else if (type === "blog" && title) {
      textHTML = `<div class="post-text"><b>${this.escapeHtml(title)}</b>${content ? `<br>${this.escapeHtml(content)}` : ""}</div>`;
    } else if (title && title !== content && title !== author) {
      textHTML = `<div class="post-text">${this.escapeHtml(title)}${content ? `<br>${this.escapeHtml(content)}` : ""}</div>`;
    } else if (content) {
      textHTML = `<div class="post-text">${this.escapeHtml(content)}</div>`;
    }

    const compact = ["like", "comment", "share", "mention", "visitor"].includes(type);
    const mediaHTML = type === "album" ? this.buildAlbumCover(item) : this.buildImagesBlock(item, compact);
    const actionHTML = this.buildPostActions(item, type);
    const deletedHTML = item.is_deleted ? `<span class="deleted-badge">已删除</span>` : "";
    const targetType = item.target_type || type;
    const targetId = item.target_id || item.id || "";

    const avatar = this.buildDeferredImageTag(
      this.proxyImageUrl(`https://q.qlogo.cn/headimg_dl?dst_uin=${authorQQ}&spec=100`),
      { className: "av timeline-avatar lazy-img" },
    );

    div.dataset.targetType = targetType;
    div.dataset.targetId = targetId;
    div.innerHTML = `
      <div class="post-head timeline-head">
        ${avatar}
        <div class="post-meta">
          <div class="nm timeline-author">${this.escapeHtml(author)}</div>
          <div class="tm timeline-time">${this.escapeHtml(time)}</div>
        </div>
      </div>
      ${textHTML}
      ${mediaHTML}
      ${actionHTML}
      <div class="interaction-detail hidden"></div>
      ${deletedHTML}`;

    return div;
  },

  buildPostActions(item, type) {
    if (type === "visitor") return "";
    const likes = Array.isArray(item.like_preview) ? item.like_preview : [];
    const comments = Array.isArray(item.comment_preview) ? item.comment_preview : [];
    if (type === "message" && !likes.length && !comments.length && !item.can_expand) return "";

    const like = Number(item.like_count || 0);
    const comment = Number(item.comment_count || 0);
    const share = Number(item.share_count || 0);
    const targetType = item.target_type || type;
    const targetId = item.target_id || item.id || "";
    const canExpand = Boolean(targetType && targetId && (like > 0 || comment > 0 || share > 0 || item.can_expand));
    const expandCall = canExpand
      ? `app.toggleMemoryInteractions(this, '${this.jsString(targetType)}', '${this.jsString(targetId)}')`
      : "";
    const labelLike = type === "like" ? "已赞" : `赞${like ? " " + this.formatCount(like) : ""}`;
    const labelComment = type === "comment" ? "评论" : `评论${comment ? " " + this.formatCount(comment) : ""}`;
    const labelShare = `转发${share ? " " + this.formatCount(share) : ""}`;
    const previewHTML = `${this.buildLikePreviewLine(item, likes, like)}${this.buildCommentPreviewBox(comments)}`;
    return `<div class="barwrap">
      <div class="bar">
        <button type="button" aria-label="赞" ${expandCall ? `onclick="${expandCall}"` : ""}>
          <svg viewBox="0 0 24 24"><path d="M12 21s-7-4.5-9.5-9C.8 8.4 2.6 5 6 5c2 0 3.2 1.2 4 2.3C10.8 6.2 12 5 14 5c3.4 0 5.2 3.4 3.5 7C19 16.5 12 21 12 21z"/></svg>${this.escapeHtml(labelLike)}
        </button>
        <button type="button" aria-label="评论" ${expandCall ? `onclick="${expandCall}"` : ""}>
          <svg viewBox="0 0 24 24"><path d="M21 12a8 8 0 0 1-11.5 7.2L3 21l1.8-6.5A8 8 0 1 1 21 12z"/></svg>${this.escapeHtml(labelComment)}
        </button>
        <button type="button" aria-label="转发" ${expandCall ? `onclick="${expandCall}"` : ""}>
          <svg viewBox="0 0 24 24"><path d="M4 9l5-5v3c7 0 11 4 11 11-2.5-4-6-5-11-5v3z"/></svg>${this.escapeHtml(labelShare)}
        </button>
      </div>
      ${previewHTML}
    </div>`;
  },

  async toggleMemoryInteractions(button, targetType, targetId) {
    const card = button?.closest?.(".timeline-item");
    const box = card?.querySelector?.(".interaction-detail");
    if (!card || !box || !targetType || !targetId) return;
    if (box.dataset.loaded === "1") {
      box.classList.toggle("hidden");
      return;
    }
    box.classList.remove("hidden");
    box.innerHTML = `<div class="interaction-loading">正在展开完整互动...</div>`;
    try {
      const res = await this.api(
        `/api/v1/memory/item/interactions?qq=${this.qq}&target_type=${encodeURIComponent(targetType)}&target_id=${encodeURIComponent(targetId)}`,
      );
      if (res.code !== 0 || !res.data) {
        box.innerHTML = `<div class="interaction-empty">暂时没有可展开的互动</div>`;
        return;
      }
      box.dataset.loaded = "1";
      box.innerHTML = this.renderInteractionDetail(res.data);
      this.observeDeferredImages(box);
    } catch {
      box.innerHTML = `<div class="interaction-empty">互动加载失败，稍后再试</div>`;
    }
  },

  renderInteractionDetail(data = {}) {
    const likes = Array.isArray(data.likes) ? data.likes : [];
    const comments = Array.isArray(data.comments) ? data.comments : [];
    const shares = Array.isArray(data.shares) ? data.shares : [];
    const sections = [];
    if (likes.length) {
      sections.push(`<section class="interaction-section">
        <h4>赞过的人 <span>${this.formatCount(likes.length)}</span></h4>
        <div class="interaction-people">${likes.map((item) => this.renderInteractionPerson(item)).join("")}</div>
      </section>`);
    }
    if (comments.length) {
      sections.push(`<section class="interaction-section">
        <h4>评论 <span>${this.formatCount(comments.length)}</span></h4>
        <div class="interaction-comments">${comments.map((item) => this.renderInteractionComment(item)).join("")}</div>
      </section>`);
    }
    if (shares.length) {
      sections.push(`<section class="interaction-section">
        <h4>转发 <span>${this.formatCount(shares.length)}</span></h4>
        <div class="interaction-comments">${shares.map((item) => this.renderInteractionShare(item)).join("")}</div>
      </section>`);
    }
    return sections.length ? sections.join("") : `<div class="interaction-empty">这条回忆还没有可展开的互动</div>`;
  },

  renderInteractionPerson(item = {}) {
    const qq = item.qq || "";
    const name = this.cleanName(item.name, qq || "好友");
    const avatar = this.buildDeferredImageTag(
      this.proxyImageUrl(item.avatar || (qq ? `https://q.qlogo.cn/headimg_dl?dst_uin=${qq}&spec=100` : "")),
      { className: "lazy-img" },
    );
    return `<button class="interaction-person" type="button" onclick="app.searchInteractionsFor('${this.jsString(qq || name)}')">
      ${avatar}
      <span>${this.escapeHtml(name)}</span>
    </button>`;
  },

  renderInteractionComment(item = {}) {
    const qq = item.qq || "";
    const name = this.cleanName(item.name, qq || "好友");
    const time = this.formatTime(item.time || "");
    const replyTo = this.cleanName(item.reply_to_name, "");
    const content = String(item.content || "").trim();
    const avatar = this.buildDeferredImageTag(
      this.proxyImageUrl(item.avatar || (qq ? `https://q.qlogo.cn/headimg_dl?dst_uin=${qq}&spec=100` : "")),
      { className: "lazy-img" },
    );
    return `<article class="interaction-comment">
      ${avatar}
      <div>
        <div class="interaction-comment-head">
          <button type="button" onclick="app.searchInteractionsFor('${this.jsString(qq || name)}')">${this.escapeHtml(name)}</button>
          <span>${this.escapeHtml(time)}</span>
        </div>
        <p>${replyTo ? `回复 ${this.escapeHtml(replyTo)}：` : ""}${this.escapeHtml(content || "评论了这条回忆")}</p>
      </div>
    </article>`;
  },

  renderInteractionShare(item = {}) {
    const qq = item.qq || "";
    const name = this.cleanName(item.name, qq || "好友");
    const time = this.formatTime(item.time || "");
    const comment = String(item.comment || "").trim();
    const avatar = this.buildDeferredImageTag(
      this.proxyImageUrl(item.avatar || (qq ? `https://q.qlogo.cn/headimg_dl?dst_uin=${qq}&spec=100` : "")),
      { className: "lazy-img" },
    );
    return `<article class="interaction-comment">
      ${avatar}
      <div>
        <div class="interaction-comment-head">
          <button type="button" onclick="app.searchInteractionsFor('${this.jsString(qq || name)}')">${this.escapeHtml(name)}</button>
          <span>${this.escapeHtml(time)}</span>
        </div>
        <p>${this.escapeHtml(comment || "转发了这条回忆")}</p>
      </div>
    </article>`;
  },

  buildLikePreviewLine(item, likes, total) {
    if (!likes.length || !total) return "";
    const avatars = likes
      .map((like) => {
        const raw = like.avatar || (like.qq ? `https://q.qlogo.cn/headimg_dl?dst_uin=${like.qq}&spec=100` : "");
        if (!raw) return "";
        return this.buildDeferredImageTag(this.proxyImageUrl(raw), { className: "lazy-img" });
      })
      .filter(Boolean)
      .join("");
    const names = likes
      .map((like) => this.cleanName(like.name, like.qq || "好友"))
      .filter(Boolean)
      .slice(0, 3);
    const text = total > names.length
      ? `${names.join("、")} 等 ${this.formatCount(total)} 人觉得很赞`
      : `${names.join("、")}觉得很赞`;
    return `<div class="likeline">
      <span class="heart">♥</span>
      ${avatars ? `<span class="avs">${avatars}</span>` : ""}
      <span>${this.escapeHtml(text)}</span>
    </div>`;
  },

  buildCommentPreviewBox(comments) {
    const rows = comments
      .filter((comment) => String(comment.content || "").trim())
      .slice(0, 2)
      .map((comment) => {
        const name = this.cleanName(comment.name, comment.qq || "好友");
        const content = String(comment.content || "").trim();
        const replyToName = this.cleanName(comment.reply_to_name, "");
        const reply = replyToName
          ? `<span class="reply">回复 ${this.escapeHtml(replyToName)}：${this.escapeHtml(content)}</span>`
          : this.escapeHtml(content);
        return `<div class="cmt"><b>${this.escapeHtml(name)}：</b>${reply}</div>`;
      })
      .join("");
    return rows ? `<div class="cmts">${rows}</div>` : "";
  },

  buildImagesBlock(item, compact = false) {
    let urls = [];
    if (item.images) {
      try {
        const arr =
          typeof item.images === "string" ? JSON.parse(item.images) : item.images;
        if (Array.isArray(arr)) urls = arr;
      } catch {}
    }
    urls = urls.filter((u) => u && !this.isDecorativeImage(u));
    if (urls.length === 0) {
      const cover = item.cover || item.preview_url || "";
      if (cover && !this.isDecorativeImage(cover)) urls = [cover];
    }
    if (urls.length === 0) return "";

    const proxied = urls.map((u) => this.proxyImageUrl(u));
    const json = JSON.stringify(proxied).replace(/"/g, "&quot;");

    if (compact) {
      const click = `app.openLightbox(${json}, 0)`.replace(/"/g, "&quot;");
      const img = this.buildDeferredImageTag(proxied[0], { className: "lazy-img" });
      const more =
        proxied.length > 1 ? `<span class="img-more">+${proxied.length - 1}</span>` : "";
      return `<div class="timeline-images compact"><div class="img-cell" onclick="${click}">${img}${more}</div></div>`;
    }

    const shown = proxied.slice(0, 3);
    const cols = Math.min(shown.length, 3);
    const cells = shown
      .map((u, i) => {
        const isLast = i === shown.length - 1;
        const more =
          isLast && proxied.length > shown.length
            ? `<span class="img-more">+${proxied.length - shown.length}</span>`
            : "";
        const img = this.buildDeferredImageTag(u, { className: "lazy-img" });
        const click = `app.openLightbox(${json}, ${i})`.replace(/"/g, "&quot;");
        return `<div class="img-cell" onclick="${click}">${img}${more}</div>`;
      })
      .join("");
    return `<div class="timeline-images cols-${cols}">${cells}</div>`;
  },

  buildAlbumCover(item) {
    const albumId = item.id || item.album_id || "";
    const name = this.cleanName(item.title || item.name, "未命名相册");
    const coverRaw = item.cover || item.cover_url || "";
    const cover =
      coverRaw && !this.isDecorativeImage(coverRaw)
        ? this.proxyImageUrl(coverRaw)
        : "";
    const img = cover ? this.buildDeferredImageTag(cover, { className: "lazy-img" }) : "";
    const hint = item.photo_count ? `${item.photo_count} 张照片` : "点击翻看相册";
    return `<div class="album-cover" data-album-id="${this.escapeHtml(albumId)}" data-album-name="${this.escapeHtml(name)}" onclick="app.openAlbumDetailFromEl(this)">
      ${img}
      <div class="album-cover-info">
        <div>
          <div class="album-cover-name">${this.escapeHtml(name)}</div>
          <div class="album-cover-hint">${this.escapeHtml(hint)}</div>
        </div>
        <span class="album-cover-go">进入 →</span>
      </div>
    </div>`;
  },

  displayAuthor(item) {
    const qq = item.author_qq || item.user_qq || "";
    const isOwner = !qq || qq === this.qq;
    if (isOwner) return this.cleanName(item.author_name || this.nickname, "我");
    return this.cleanName(item.author_name, qq || "好友");
  },

  detectType(item) {
    if (item.type) return item.type;
    if (item.talk_id) return "talk";
    if (item.blog_id) return "blog";
    if (item.album_id && !item.photo_id) return "album";
    if (item.message_id) return "message";
    if (item.mention_id) return "mention";
    if (item.share_id) return "share";
    return "other";
  },

  renderFriends(data) {
    const filterContainer = document.getElementById("friend-group-filters");
    const groupsContainer = document.getElementById("friend-groups");
    const sourceList = data?.list || [];
    const groups = data?.groups || [];
    const activeGroups = groups.filter((item) => item && !item.is_deleted);
    const groupMap = new Map(
      activeGroups.map((group) => [String(group.group_id), group.name || "未分组"]),
    );
    const discoveredGroupIds = new Set();
    const list = sourceList.filter((item) => {
      const isHistorical = item.is_deleted || !item.is_current;
      if (this.friendFilter === "current" && isHistorical) return false;
      if (this.friendFilter === "historical" && !isHistorical) return false;
      const groupKey = String(item.group_id);
      discoveredGroupIds.add(groupKey);
      if (this.friendGroupFilter !== "all" && groupKey !== this.friendGroupFilter) {
        return false;
      }
      if (!this.friendSearch) return true;
      const keyword = this.friendSearch.toLowerCase();
      return [
        item.name,
        item.remark,
        item.friend_qq,
        item.group_name,
      ].some((value) => String(value || "").toLowerCase().includes(keyword));
    });
    const availableGroupIds = Array.from(discoveredGroupIds);
    if (this.friendGroupFilter !== "all" && !availableGroupIds.includes(this.friendGroupFilter)) {
      this.friendGroupFilter = "all";
    }
    const filterOptions = [
      { id: "all", name: "全部分组", count: sourceList.length },
      ...availableGroupIds
        .sort((a, b) => Number(a) - Number(b))
        .map((groupId) => ({
          id: groupId,
          name: groupMap.get(groupId) || "未分组",
          count: sourceList.filter((item) => String(item.group_id) === groupId).length,
        })),
    ];
    filterContainer.innerHTML = filterOptions
      .map((group) => `
        <button
          class="chip ${this.friendGroupFilter === group.id ? "active" : ""}"
          onclick="app.switchFriendGroupFilter('${this.escapeHtml(group.id)}', event)"
        >${this.escapeHtml(group.name)}${group.id === "all" ? "" : ` · ${group.count}`}</button>
      `)
      .join("");
    document
      .querySelectorAll("#friend-status-filters .chip")
      .forEach((el) => el.classList.toggle("active", el.dataset.friendFilter === this.friendFilter));

    const grouped = new Map();
    for (const group of activeGroups) {
      grouped.set(String(group.group_id), {
        group,
        items: [],
      });
    }
    for (const friend of list) {
      const key = String(friend.group_id);
      if (!grouped.has(key)) {
        grouped.set(key, {
          group: {
            group_id: friend.group_id,
            name: friend.group_name || "未分组",
            is_deleted: false,
          },
          items: [],
        });
      }
      grouped.get(key).items.push(friend);
    }

    const sections = Array.from(grouped.values())
      .filter((entry) => entry.items.length > 0)
      .sort((a, b) => a.group.group_id - b.group.group_id)
      .map((entry) => {
        const cards = entry.items
          .sort((a, b) => {
            if (a.is_current !== b.is_current) return a.is_current ? -1 : 1;
            return (b.interact_count || 0) - (a.interact_count || 0);
          })
          .map((friend) => this.renderFriendCard(friend))
          .join("");

        return `<section class="friend-group-section">
          <div class="friend-group-header">
            <h3>${this.escapeHtml(entry.group.name || "未分组")}</h3>
            <span>${entry.items.length} 人</span>
          </div>
          <div class="friend-card-grid">${cards}</div>
        </section>`;
      })
      .join("");

    groupsContainer.innerHTML = sections;
    this.observeDeferredImages(groupsContainer);
  },

  renderFriendCard(friend) {
    const isHistorical = friend.is_deleted || !friend.is_current;
    const name = this.cleanName(friend.name || friend.remark, friend.friend_qq || "好友");
    const remark = this.cleanName(friend.remark, "");
    const sub = remark && remark !== name ? remark : "QQ " + (friend.friend_qq || "");
    const badgeClass = isHistorical ? "historical" : "current";
    const badgeText = isHistorical ? "旧联系人" : "好友";
    const avatar = this.buildDeferredImageTag(
      this.proxyImageUrl(
        friend.avatar || `https://q.qlogo.cn/headimg_dl?dst_uin=${friend.friend_qq}&spec=100`,
      ),
      { className: "friend-avatar lazy-img" },
    );

    const chips = [];
    if (friend.is_special_care) chips.push(`<span class="friend-chip care">★ 特别关心</span>`);
    if (friend.interact_count > 0) chips.push(`<span class="friend-chip">互动 ${this.formatCount(friend.interact_count)}</span>`);
    if (friend.yellow > 0) chips.push(`<span class="friend-chip">黄钻 ${friend.yellow}</span>`);

    const searchKey = friend.friend_qq || name;
    return `<article class="friend-card${isHistorical ? " historical" : ""}" onclick="app.searchInteractionsFor('${this.jsString(searchKey)}')">
      <div class="friend-card-head">
        ${avatar}
        <div>
          <div class="friend-name">${this.escapeHtml(name)}</div>
          <div class="friend-remark">${this.escapeHtml(sub)}</div>
        </div>
        <span class="friend-badge ${badgeClass}">${badgeText}</span>
      </div>
      ${chips.length ? `<div class="friend-card-body">${chips.join("")}</div>` : ""}
    </article>`;
  },

  handleFriendSearch(event) {
    this.friendSearch = (event.target.value || "").trim();
    this.loadData();
  },

  switchFriendFilter(filter, event) {
    if (event) event.preventDefault();
    this.friendFilter = filter || "all";
    this.loadData();
  },

  switchFriendGroupFilter(groupId, event) {
    if (event) event.preventDefault();
    this.friendGroupFilter = groupId || "all";
    this.loadData();
  },

  // ===== 同步 =====
  async startSync() {
    if (this.syncStarting) return;
    this.syncStarting = true;
    this.setSyncButtonState(true);
    try {
      this.updateSyncStatus("启动中", "正在创建同步任务，请稍候");
      const res = await this.api("/api/v1/sync/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ qq: this.qq }),
      });

      if (res.code === 0) {
        this.showSyncProgress();
      } else {
        this.setSyncButtonState(false);
        if (this.handleAuthExpired(res.message, res.code)) return;
        alert(res.message || "请求失败");
      }
    } catch (e) {
      this.setSyncButtonState(false);
      alert(e.message || "请求失败");
    } finally {
      this.syncStarting = false;
    }
  },

  showSyncProgress() {
    document.getElementById("empty-state").classList.add("hidden");
    const loadingState = document.getElementById("loading-state");
    if (loadingState) loadingState.classList.add("hidden");
    document.getElementById("sync-progress").classList.remove("hidden");
    this.updateSyncControls("running");
    this.updateSyncStatus("同步中", "正在从 QQ 空间拉取并整理数据", "running");
    this.setSyncButtonState(true);

    this.startSyncPoll();
  },

  startSyncPoll() {
    this.stopSyncPoll();
    this.syncTimer = setInterval(() => this.pollSyncProgress(), 1500);
  },

  stopSyncPoll() {
    if (this.syncTimer) {
      clearInterval(this.syncTimer);
      this.syncTimer = null;
    }
  },

  async pollSyncProgress() {
    try {
      const res = await this.api("/api/v1/sync/progress");
      if (res.code !== 0) return;

      const data = res.data;
      const pct =
        data.total_types > 0
          ? Math.round((data.done_types / data.total_types) * 100)
          : 0;
      document.getElementById("sync-progress-bar").style.width = pct + "%";
      document.getElementById("sync-progress-text").textContent =
        data.message ||
        `${data.current_type || "准备中"} (${data.done_types}/${data.total_types})`;
      this.applySyncStatus(data);

      if (data.status === "paused") {
        this.stopSyncPoll();
        this.setSyncButtonState(false);
        this.updateSyncControls("paused");
        this.loadData({ resetScroll: false });
        return;
      }

      if (data.status === "done" || data.status === "error" || data.status === "idle") {
        this.stopSyncPoll();
        document.getElementById("sync-progress").classList.add("hidden");
        this.updateSyncControls("stopped");
        this.setSyncButtonState(false);

        if (data.status === "error") {
          this.updateSyncStatus("同步失败", data.error || "同步失败", "error");
          if (this.handleAuthExpired(data.error)) return;
          alert(data.error || "同步失败");
        } else if (data.status === "done") {
          this.updateSyncStatus("同步完成", "数据已是最新，可以继续翻阅", "done");
        }

        // 刷新数据
        this.loadData({ resetScroll: false });
      }
    } catch {}
  },

  // ===== 同步弹窗 =====
  showSyncDialog() {
    document.getElementById("sync-modal").classList.remove("hidden");
  },

  closeModal(modalId = "sync-modal") {
    const modal = document.getElementById(modalId);
    if (modal) modal.classList.add("hidden");
  },

  confirmSync() {
    this.closeModal("sync-modal");
    this.startSync();
  },

  // ===== 同步控制（暂停 / 继续 / 取消）=====
  updateSyncControls(state) {
    const box = document.getElementById("sync-controls");
    const toggle = document.getElementById("btn-sync-toggle");
    if (!box || !toggle) return;
    if (state === "running") {
      box.classList.remove("hidden");
      toggle.textContent = "暂停";
      toggle.dataset.action = "pause";
    } else if (state === "paused") {
      box.classList.remove("hidden");
      toggle.textContent = "继续";
      toggle.dataset.action = "resume";
    } else {
      box.classList.add("hidden");
    }
  },

  toggleSyncPause() {
    const toggle = document.getElementById("btn-sync-toggle");
    if (toggle && toggle.dataset.action === "resume") {
      this.resumeSync();
    } else {
      this.pauseSync();
    }
  },

  async pauseSync() {
    try {
      await this.api("/api/v1/sync/pause", { method: "POST" });
    } catch {}
  },

  async resumeSync() {
    try {
      const res = await this.api("/api/v1/sync/resume", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ qq: this.qq }),
      });
      if (res.code === 0) this.showSyncProgress();
    } catch {}
  },

  async cancelSync() {
    try {
      await this.api("/api/v1/sync/cancel", { method: "POST" });
    } catch {}
    this.stopSyncPoll();
    document.getElementById("sync-progress").classList.add("hidden");
    this.updateSyncControls("stopped");
    this.setSyncButtonState(false);
    this.updateSyncStatus("待同步", "从 QQ 空间拉取并整理你的历史数据", "");
  },

  // ===== 数据与隐私 =====
  openPrivacy() {
    document.getElementById("privacy-modal").classList.remove("hidden");
    this.loadStorageStats();
  },

  async loadStorageStats() {
    try {
      const res = await this.api(`/api/v1/storage/stats?qq=${this.qq}`);
      if (res.code === 0 && res.data) this.renderStorageStats(res.data);
    } catch {}
  },

  renderStorageStats(s) {
    const setText = (id, value) => {
      const el = document.getElementById(id);
      if (el) el.textContent = value;
    };
    setText("privacy-db-path", s.db_path || "—");
    setText("privacy-media-dir", s.media_dir || "—");

    const by = s.media_by_status || {};
    const done = by.done || 0;
    const pending = by.pending || 0;
    const failed = by.failed || 0;
    const mb = ((s.media_bytes || 0) / 1048576).toFixed(1);
    setText(
      "privacy-media-stat",
      `已存 ${done} · 待下载 ${pending} · 失败 ${failed} · 占用 ${mb} MB`,
    );

    const badge = document.getElementById("privacy-badge");
    if (badge) badge.classList.toggle("hidden", !(done > 0 && pending === 0 && failed === 0));
  },

  async backfillMedia() {
    try {
      await this.api(`/api/v1/media/backfill?qq=${this.qq}`, { method: "POST" });
      alert("已在后台开始下载，稍后重新打开本面板可看到进度");
    } catch {
      alert("操作失败");
    }
  },

  confirmDeleteAll() {
    if (!window.confirm("确定要彻底删除本机上这个账号的全部数据、图片与登录态吗？此操作不可恢复。")) return;
    if (!window.confirm("再次确认：删除后需要重新扫码登录并同步。继续？")) return;
    this.deleteAllData();
  },

  async deleteAllData() {
    try {
      const res = await this.api("/api/v1/data/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ qq: this.qq, confirm: true }),
      });
      if (res.code === 0) {
        alert("数据已彻底删除");
        this.closeModal("privacy-modal");
        this.logout();
      } else {
        alert(res.message || "删除失败");
      }
    } catch {
      alert("删除失败");
    }
  },

  // ===== 导出纪念册 =====
  exportArchive() {
    if (!this.qq) return;
    // 触发浏览器下载（后端返回带 Content-Disposition 的 zip）
    window.location.href = `/api/v1/export?qq=${encodeURIComponent(this.qq)}`;
  },

  // ===== 年度纪念卡 =====
  openReport() {
    document.getElementById("report-modal").classList.remove("hidden");
    const body = document.getElementById("report-body");
    if (body) body.innerHTML = '<p class="rail-placeholder">正在生成…</p>';
    this.loadReport();
  },

  async loadReport() {
    try {
      const res = await this.api(`/api/v1/memory/report?qq=${this.qq}`);
      if (res.code === 0 && res.data) this.renderReport(res.data);
    } catch {}
  },

  renderReport(r) {
    const body = document.getElementById("report-body");
    if (!body) return;
    const fmt = (t) => {
      if (!t) return "—";
      const d = new Date(t);
      if (isNaN(d.getTime())) return "—";
      return `${d.getFullYear()}.${d.getMonth() + 1}.${d.getDate()}`;
    };
    const labels = this.typeLabels;
    const byType = r.by_type || {};
    const typeRows = Object.entries(byType)
      .filter(([, v]) => v > 0)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 6)
      .map(([t, c]) => `<li>${this.escapeHtml(labels[t] || t)} · ${this.formatCount(c)}</li>`)
      .join("");
    const people = (r.top_people || [])
      .map((p) => `<li>${this.escapeHtml(this.cleanName(p.name, p.qq))} · 互动 ${this.formatCount(p.count)}</li>`)
      .join("");
    const firstTalk = r.first_talk
      ? `<p>你的第一条说说（${fmt(r.first_talk.publish_time)}）：<br/>「${this.escapeHtml((r.first_talk.content || "").slice(0, 80)) || "（无文字）"}」</p>`
      : "";

    body.innerHTML = `
      <p>你在 QQ 空间共留下了 <strong>${this.formatCount(r.total || 0)}</strong> 条回忆，横跨 <strong>${r.span_years || 0}</strong> 年（${fmt(r.first_time)} — ${fmt(r.last_time)}）。</p>
      <p>最活跃的是 <strong>${r.most_active_year || "—"}</strong> 年，那一年你留下了 ${this.formatCount(r.most_active_year_count || 0)} 条。</p>
      ${people ? `<p>这些年陪你互动最多的人：</p><ul class="modal-list">${people}</ul>` : ""}
      <p>回忆构成：</p>
      <ul class="modal-list">${typeRows || "<li>—</li>"}</ul>
      ${firstTalk}`;
  },

  isAuthExpired(message = "", code) {
    const text = String(message || "");
    return (
      code === 401 ||
      text.includes("Cookie 已过期") ||
      text.includes("Cookie 无效") ||
      text.includes("授权失败") ||
      text.includes("请重新登录")
    );
  },

  handleAuthExpired(message = "", code) {
    if (!this.isAuthExpired(message, code)) return false;

    const shouldRelogin = window.confirm(
      `${message || "当前登录状态已失效"}\n\n是否立即重新登录？`,
    );
    if (shouldRelogin) {
      this.logout();
    }
    return true;
  },

  // ===== 工具函数 =====
  async api(url, options = {}) {
    const res = await fetch(url, options);
    return await res.json();
  },

  sleep(ms) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  },

  formatTime(timeStr) {
    if (!timeStr) return "";
    const d = new Date(timeStr);
    if (isNaN(d.getTime())) return timeStr;
    const now = new Date();
    const diff = now - d;

    if (diff < 60000) return "刚刚";
    if (diff < 3600000) return Math.floor(diff / 60000) + "分钟前";
    if (diff < 86400000) return Math.floor(diff / 3600000) + "小时前";

    const y = d.getFullYear();
    const m = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    const h = String(d.getHours()).padStart(2, "0");
    const min = String(d.getMinutes()).padStart(2, "0");

    if (y === now.getFullYear()) return `${m}-${day} ${h}:${min}`;
    return `${y}-${m}-${day} ${h}:${min}`;
  },

  jsString(str) {
    return String(str || "")
      .replace(/\\/g, "\\\\")
      .replace(/'/g, "\\'")
      .replace(/\n/g, " ");
  },

  escapeHtml(str) {
    if (!str) return "";
    const div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  },

  proxyImageUrl(url) {
    if (!url) return "";
    if (url.includes(".qq.com") || url.includes(".qlogo.cn") || url.includes(".qpic.cn")) {
      // 带上 qq，命中本地的资源会由后端直接返回本地文件，浏览逐步走向零外部请求
      const qq = this.qq ? "&qq=" + encodeURIComponent(this.qq) : "";
      return "/api/v1/proxy/image?url=" + encodeURIComponent(url) + qq;
    }
    return url;
  },

  // 过滤头像、QQ 空间装饰素材与表情贴纸，只保留真实照片
  isDecorativeImage(url) {
    const v = String(url || "").trim().toLowerCase();
    if (!v) return true;
    if (v.includes("q.qlogo.cn") || v.includes("/headimg") || v.includes("avatar")) return true;
    if (v.includes("qzonestyle.gtimg.cn") || v.includes("/space_item/") || v.includes("custompraise")) return true;
    if (v.includes("/emotion/") || v.includes("/qzone/em/") || v.includes("/com_attr/")) return true;
    if (/\/qzone\/\d+\/\d+\/\d+/.test(v)) return true; // 头像缩略图路径
    return false;
  },

  // 清理昵称中的零宽 / 不可见字符，必要时回退
  cleanName(str, fallback = "") {
    let out = "";
    for (const ch of String(str || "")) {
      const c = ch.codePointAt(0);
      const invisible =
        (c >= 0x200b && c <= 0x200f) ||
        (c >= 0x2028 && c <= 0x202f) ||
        (c >= 0x2060 && c <= 0x2064) ||
        c === 0xfeff;
      if (!invisible) out += ch;
    }
    out = out.trim();
    return out || fallback;
  },

  // 大数缩写：1.2w / 3.4k
  formatCount(value) {
    const num = Number(value) || 0;
    if (num >= 10000) return (num / 10000).toFixed(num % 10000 === 0 ? 0 : 1) + "w";
    if (num >= 1000) return (num / 1000).toFixed(num % 1000 === 0 ? 0 : 1) + "k";
    return String(num);
  },

  updateSyncStatus(label, hint, state) {
    const stateEl = document.getElementById("sync-state");
    const hintEl = document.getElementById("sync-hint");
    if (stateEl) {
      stateEl.textContent = label;
      stateEl.classList.remove("is-running", "is-done", "is-error");
      if (state === "running") stateEl.classList.add("is-running");
      else if (state === "done") stateEl.classList.add("is-done");
      else if (state === "error") stateEl.classList.add("is-error");
    }
    if (hintEl) hintEl.textContent = hint || "";
  },

  applySyncStatus(data = {}) {
    const status = data.status || "idle";
    switch (status) {
      case "running":
        this.updateSyncStatus("同步中", data.message || "正在从 QQ 空间拉取并整理数据", "running");
        break;
      case "done":
        this.updateSyncStatus("同步完成", data.message || "数据已是最新，可以继续翻阅", "done");
        break;
      case "error":
        this.updateSyncStatus("同步失败", data.error || data.message || "最近一次同步执行失败", "error");
        break;
      case "paused":
        this.updateSyncStatus("已暂停", data.message || "同步已暂停，可继续", "running");
        break;
      case "idle":
      default:
        this.updateSyncStatus("待同步", "从 QQ 空间拉取并整理你的历史数据", "");
        break;
    }
  },

  updateUnifiedSearchField() {
    const input = document.getElementById("memory-search");
    if (!input) return;

    input.placeholder = "搜索说说、留言、好友昵称或 QQ…";
    if (document.activeElement !== input) {
      input.value = this.currentView === "interactions" ? this.interactionKeyword || "" : this.searchKeyword || "";
    }
  },

  updateContentHead() {
    const meta = this.viewMeta[this.currentView] || this.viewMeta.memories;
    document.getElementById("content-kicker").textContent = meta.kicker;
    document.getElementById("content-title").textContent = meta.title;
    document.getElementById("content-description").textContent =
      meta.description;
    if (this.currentView === "memories" && this.searchKeyword) {
      document.getElementById("content-kicker").textContent = "搜索";
      document.getElementById("content-title").textContent = `搜索 “${this.searchKeyword}”`;
      document.getElementById("content-description").textContent = "在全部回忆中按关键词检索";
    }
    if (this.currentView === "interactions" && this.interactionKeyword) {
      document.getElementById("content-kicker").textContent = "好友互动";
      document.getElementById("content-title").textContent = `与 “${this.interactionKeyword}” 的互动`;
      document.getElementById("content-description").textContent = "点赞、评论、留言、访客与内容提及会按时间汇总在这里";
    }
  },

  formatTimelineGroup(timeStr) {
    if (!timeStr) return "earlier";
    const d = new Date(timeStr);
    if (isNaN(d.getTime())) return "earlier";
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
  },

  updateEmptyState() {
    const meta = this.viewMeta[this.currentView] || this.viewMeta.memories;
    document.getElementById("empty-title").textContent = meta.emptyTitle;
    document.getElementById("empty-description").textContent =
      meta.emptyDescription;
    if (this.currentView === "memories" && this.searchKeyword) {
      document.getElementById("empty-title").textContent = "没有匹配的回忆";
      document.getElementById("empty-description").textContent = "换个关键词再试试";
    }
    if (this.currentView === "interactions") {
      document.getElementById("empty-title").textContent = this.interactionKeyword ? "没有找到互动" : "先输入一个好友";
      document.getElementById("empty-description").textContent = this.interactionKeyword ? "换个昵称或 QQ 号再试试" : "在右上角搜索框输入昵称或 QQ，查看你和 TA 的所有往来";
    }
  },

  // ===== 相册详情 =====
  async openAlbumDetail(albumId, albumName) {
    this.albumDetailOpen = true;
    this.albumDetailId = albumId;
    this.albumDetailName = albumName;
    this.albumPhotoPage = 1;

    document.getElementById("timeline").classList.add("hidden");
    document.getElementById("load-more").classList.add("hidden");
    document.getElementById("empty-state").classList.add("hidden");
    document.getElementById("album-detail").classList.remove("hidden");
    document.getElementById("album-detail-title").textContent = albumName;
    document.getElementById("album-photos").innerHTML = "";
    document.getElementById("album-detail-count").textContent = "加载中";
    document.getElementById("album-load-more").classList.add("hidden");

    await this.fetchAlbumPhotos();
  },

  openAlbumDetailFromEl(el) {
    if (!el) return;
    this.openAlbumDetail(el.dataset.albumId || "", el.dataset.albumName || "相册");
  },

  closeAlbumDetail() {
    if (!this.albumDetailOpen) return;
    this.albumDetailOpen = false;
    document.getElementById("album-detail").classList.add("hidden");
    // 重新显示相册列表
    document.getElementById("timeline").classList.remove("hidden");
    if (this.page > 0) {
      const items = document.querySelectorAll("#timeline .timeline-item");
      if (items.length >= this.page * this.pageSize) {
        document.getElementById("load-more").classList.remove("hidden");
      }
    }
  },

  async fetchAlbumPhotos() {
    try {
      const res = await this.api(
        `/api/v1/photos/by-album?album_id=${this.albumDetailId}&qq=${this.qq}&page=${this.albumPhotoPage}&page_size=30`,
      );
      if (res.code === 0 && res.data) {
        const list = res.data.list || [];
        const total = res.data.total || 0;
        document.getElementById("album-detail-count").textContent =
          `${total} 张照片`;
        this.renderAlbumPhotos(list);
        if (list.length >= 30) {
          document.getElementById("album-load-more").classList.remove("hidden");
        } else {
          document.getElementById("album-load-more").classList.add("hidden");
        }
      }
    } catch {}
  },

  renderAlbumPhotos(photos) {
    const container = document.getElementById("album-photos");
    photos.forEach((photo) => {
      const full = photo.url || photo.origin_url || "";
      if (!full) return;
      const fullProxied = this.proxyImageUrl(full);
      const thumbProxied = this.proxyImageUrl(photo.thumb_url || full);
      const desc = this.cleanName(photo.description || photo.desc, "");
      const img = this.buildDeferredImageTag(thumbProxied, { className: "lazy-img" });
      const item = document.createElement("div");
      item.className = "photo-item";
      item.innerHTML = `
        <div class="photo-frame" data-album-photo data-fullsrc="${this.escapeHtml(fullProxied)}" onclick="app.openAlbumLightbox(this)">
          ${img}
        </div>
        ${desc ? `<div class="photo-desc">${this.escapeHtml(desc)}</div>` : ""}`;
      container.appendChild(item);
    });
    this.observeDeferredImages(container);
  },

  async loadMorePhotos() {
    this.albumPhotoPage++;
    await this.fetchAlbumPhotos();
  },

  // ===== 灯箱 =====
  openLightbox(images, index) {
    this.lightboxImages = images;
    this.lightboxIndex = index;
    this.updateLightbox();
    document.getElementById("lightbox").classList.remove("hidden");
  },

  openAlbumLightbox(imgEl) {
    const allImgs = Array.from(document.querySelectorAll("[data-album-photo]"));
    const urls = allImgs.map(
      (img) => img.dataset.fullsrc || img.dataset.src || img.currentSrc || img.src,
    );
    const index = allImgs.indexOf(imgEl);
    this.openLightbox(urls, index >= 0 ? index : 0);
  },

  closeLightbox(event) {
    if (event && event.target && event.target.id === "lightbox-img") return;
    document.getElementById("lightbox").classList.add("hidden");
  },

  lightboxPrev(event) {
    if (event) event.stopPropagation();
    if (this.lightboxImages.length === 0) return;
    this.lightboxIndex = (this.lightboxIndex - 1 + this.lightboxImages.length) % this.lightboxImages.length;
    this.updateLightbox();
  },

  lightboxNext(event) {
    if (event) event.stopPropagation();
    if (this.lightboxImages.length === 0) return;
    this.lightboxIndex = (this.lightboxIndex + 1) % this.lightboxImages.length;
    this.updateLightbox();
  },

  updateLightbox() {
    const img = document.getElementById("lightbox-img");
    const counter = document.getElementById("lightbox-counter");
    img.src = this.lightboxImages[this.lightboxIndex] || "";
    if (this.lightboxImages.length > 1) {
      counter.textContent = `${this.lightboxIndex + 1} / ${this.lightboxImages.length}`;
      counter.style.display = "";
    } else {
      counter.style.display = "none";
    }
  },

  setSyncButtonState(loading) {
    const syncButton = document.getElementById("btn-sync");
    const syncConfirmButton = document.getElementById("btn-sync-confirm");
    if (syncButton) {
      syncButton.disabled = loading;
      syncButton.textContent = loading ? "同步中…" : "同步档案";
    }
    if (syncConfirmButton) {
      syncConfirmButton.disabled = loading;
      syncConfirmButton.textContent = loading ? "同步中" : "确认同步";
    }
  },

  buildDeferredImageTag(src, options = {}) {
    const safeSrc = this.escapeHtml(src || "");
    if (!safeSrc) return "";

    const className = options.className
      ? ` class="${this.escapeHtml(options.className)}"`
      : "";
    const alt = options.alt ? ` alt="${this.escapeHtml(options.alt)}"` : ' alt=""';
    const onclick = options.onclick
      ? ` onclick="${options.onclick.replace(/"/g, "&quot;")}"`
      : "";
    const dataAttrs = Object.entries(options.dataAttrs || {})
      .map(([key, value]) => ` data-${key}="${this.escapeHtml(String(value))}"`)
      .join("");

    return `<img src="${this.imagePlaceholder}" data-src="${safeSrc}" loading="lazy" decoding="async" onerror="this.removeAttribute('src');this.classList.add('is-failed','is-loaded')"${className}${alt}${onclick}${dataAttrs}>`;
  },

  ensureDeferredImageObserver() {
    if (this.deferredImageObserver) return this.deferredImageObserver;
    const root = document.getElementById("content-scroll") || null;
    this.deferredImageObserver = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (!entry.isIntersecting) return;
        this.loadDeferredImage(entry.target);
        this.deferredImageObserver.unobserve(entry.target);
      });
    }, { root, rootMargin: "300px 0px" });
    return this.deferredImageObserver;
  },

  observeDeferredImages(container = document) {
    const images = container.querySelectorAll("img[data-src]");
    if (!images.length) return;

    if (!("IntersectionObserver" in window)) {
      images.forEach((img) => this.loadDeferredImage(img));
      return;
    }

    const observer = this.ensureDeferredImageObserver();
    images.forEach((img) => {
      if (img.dataset.loaded === "true") return;
      observer.observe(img);
    });
  },

  loadDeferredImage(img) {
    if (!img || img.dataset.loaded === "true") return;
    const src = img.dataset.src;
    if (!src) return;
    img.addEventListener(
      "load",
      () => {
        img.classList.add("is-loaded");
      },
      { once: true },
    );
    img.src = src;
    img.dataset.loaded = "true";
  },

  setupInfiniteLoad() {
    if (this.infiniteObserver) {
      this.infiniteObserver.disconnect();
    }
    const target = document.getElementById("load-more");
    const root = document.getElementById("content-scroll");
    if (!target || !["memories", "interactions"].includes(this.currentView)) return;
    this.infiniteObserver = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting && !this.loading && !target.classList.contains("hidden")) {
          this.loadMore();
        }
      }
    }, { root: root || null, rootMargin: "240px 0px" });
    this.infiniteObserver.observe(target);
  },
};

// 启动
document.addEventListener("DOMContentLoaded", () => app.init());
document.addEventListener("keydown", (event) => {
  const lightbox = document.getElementById("lightbox");
  const lightboxOpen = lightbox && !lightbox.classList.contains("hidden");
  if (event.key === "Escape") {
    if (lightboxOpen) {
      app.closeLightbox();
    } else {
      app.closeModal("sync-modal");
    }
  } else if (lightboxOpen && event.key === "ArrowLeft") {
    app.lightboxPrev();
  } else if (lightboxOpen && event.key === "ArrowRight") {
    app.lightboxNext();
  }
});
