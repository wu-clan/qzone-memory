// QQ 空间回忆 - 前端应用
const app = {
  qq: "",
  nickname: "",
  currentView: "memories",
  currentMemoryFilter: "all",
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
  friendFilter: "current",
  friendGroupFilter: "all",
  lastTimelineGroup: "",
  imagePlaceholder:
    "data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==",
  viewMeta: {
    memories: {
      kicker: "回忆时间线",
      title: "完整回忆",
      description: "按时间顺序回看 QQ 空间现有内容、历史动态和已删除痕迹",
      emptyTitle: "暂无回忆内容",
      emptyDescription: "当前还没有可展示的归档数据，请先执行同步",
    },
    friends: {
      kicker: "好友关系",
      title: "好友与历史联系人",
      description: "查看当前好友、分组、特别关心，以及由历史互动反推的旧联系人",
      emptyTitle: "暂无好友数据",
      emptyDescription: "当前还没有可展示的好友或历史联系人数据，请先执行同步",
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
    const avatar = document.getElementById("user-avatar");
    avatar.src = `https://q.qlogo.cn/headimg_dl?dst_uin=${this.qq}&spec=100`;
    avatar.alt = `${this.nickname || this.qq} 的头像`;
    document.getElementById("user-name").textContent = this.nickname || this.qq;

    this.loadData();
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
      status.className = "login-input-status hidden";
      return;
    }

    status.textContent = text;
    status.className = "login-input-status";
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
  async loadData() {
    this.page = 1;
    this.closeAlbumDetail();
    const timeline = document.getElementById("timeline");
    const friendsView = document.getElementById("friends-view");
    const empty = document.getElementById("empty-state");
    const loadMore = document.getElementById("load-more");
    const contentScroll = document.getElementById("content-scroll");
    const memorySummary = document.getElementById("memory-summary");

    if (contentScroll) contentScroll.scrollTop = 0;

    timeline.classList.add("hidden");
    friendsView.classList.add("hidden");
    empty.classList.add("hidden");
    loadMore.classList.add("hidden");
    document.getElementById("memory-filters").classList.toggle(
      "hidden",
      this.currentView !== "memories",
    );
    if (memorySummary) {
      memorySummary.classList.toggle("hidden", this.currentView !== "memories");
    }
    document.getElementById("friend-toolbar").classList.toggle(
      "hidden",
      this.currentView !== "friends",
    );
    this.updateContentHead();
    this.updateSyncStatus("检查中", "正在确认是否存在进行中的同步任务");
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
      this.updateSyncStatus("状态未知", "暂时无法获取同步任务状态");
    }

    const payload = await this.fetchCurrentViewData();
    const items = Array.isArray(payload) ? payload : payload?.list || [];

    if (!items || items.length === 0) {
      empty.classList.remove("hidden");
      this.updateEmptyState();
    } else {
      if (this.currentView === "friends") {
        this.renderFriends(payload);
        friendsView.classList.remove("hidden");
      } else {
        this.renderTimeline(items);
        timeline.classList.remove("hidden");
        if (items.length >= this.pageSize) {
          loadMore.classList.remove("hidden");
        }
        this.setupInfiniteLoad();
      }
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
    } else {
      url = `/api/v1/memory/timeline?qq=${qq}&type=${encodeURIComponent(this.currentMemoryFilter)}&page=${p}&page_size=${ps}`;
    }

    try {
      const res = await this.api(url);
      if (res.code !== 0 || !res.data) return this.currentView === "friends" ? null : [];
      if (this.currentView === "friends") return res.data;
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
          `/api/v1/friends?qq=${qq}&page=${page}&page_size=${pageSize}`,
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

  async loadMore() {
    if (this.loading || this.currentView !== "memories") return;
    this.loading = true;
    const loadMoreBtn = document.querySelector("#load-more .btn");
    if (loadMoreBtn) {
      loadMoreBtn.disabled = true;
      loadMoreBtn.textContent = "加载中";
    }
    this.page++;

    const items = await this.fetchCurrentViewData();
    if (items && items.length > 0) {
      this.appendTimeline(items);
      if (items.length < this.pageSize) {
        document.getElementById("load-more").classList.add("hidden");
      }
    } else {
      document.getElementById("load-more").classList.add("hidden");
    }
    this.setupInfiniteLoad();
    if (loadMoreBtn) {
      loadMoreBtn.disabled = false;
      loadMoreBtn.textContent = "加载更多";
    }
    this.loading = false;
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

    const memoriesEl = document.getElementById("count-memories");
    const friendsEl = document.getElementById("count-friends");
    if (memoriesEl) memoriesEl.textContent = memoryCount > 0 ? memoryCount : "";
    if (friendsEl) friendsEl.textContent = friendCount > 0 ? friendCount : "";
    this.updateOverviewStats(memoryCount + friendCount);
  },

  async loadMemoryStats() {
    try {
      const res = await this.api(`/api/v1/memory/stats?qq=${this.qq}`);
      if (res.code !== 0 || !res.data) return;
      const memorySummary = document.getElementById("memory-summary");
      if (memorySummary) memorySummary.innerHTML = "";
    } catch {}
  },

  switchView(view, event) {
    if (event) event.preventDefault();
    document
      .querySelectorAll(".nav-item")
      .forEach((el) => el.classList.remove("active"));
    const clicked = document.querySelector(`.nav-item[data-view="${view}"]`);
    if (clicked) clicked.classList.add("active");
    this.currentView = view;
    this.page = 1;
    this.loadData();
  },

  switchMemoryFilter(filter, event) {
    if (event) event.preventDefault();
    this.currentMemoryFilter = filter;
    document
      .querySelectorAll("#memory-filters .filter-chip")
      .forEach((el) => el.classList.toggle("active", el.dataset.filter === filter));
    this.loadData();
  },

  // ===== 时间线渲染 =====
  renderTimeline(items) {
    const container = document.getElementById("timeline");
    container.innerHTML = "";
    let currentGroup = "";
    let currentSection = null;
    items.forEach((item) => {
      const group = this.formatTimelineGroup(
        item.publish_time ||
          item.message_time ||
          item.mention_time ||
          item.create_time ||
          item.created_at,
      );
      if (group !== currentGroup) {
        currentGroup = group;
        currentSection = this.createTimelineSection(group);
        container.appendChild(currentSection);
      }
      currentSection
        .querySelector(".timeline-section-items")
        .appendChild(this.createTimelineItem(item));
    });
    this.lastTimelineGroup = currentGroup;
    this.observeDeferredImages(container);
  },

  appendTimeline(items) {
    const container = document.getElementById("timeline");
    let lastGroup = this.lastTimelineGroup || "";
    let currentSection = container.lastElementChild;
    items.forEach((item) => {
      const group = this.formatTimelineGroup(
        item.publish_time ||
          item.message_time ||
          item.mention_time ||
          item.create_time ||
          item.created_at,
      );
      if (group !== lastGroup) {
        lastGroup = group;
        currentSection = this.createTimelineSection(group);
        container.appendChild(currentSection);
      }
      currentSection
        .querySelector(".timeline-section-items")
        .appendChild(this.createTimelineItem(item));
    });
    this.lastTimelineGroup = lastGroup;
    this.observeDeferredImages(container);
  },

  createTimelineSection(label) {
    const div = document.createElement("div");
    div.className = "timeline-section";
    div.dataset.group = label;
    div.innerHTML = `
      <div class="timeline-group" data-group="${this.escapeHtml(label)}">
        <span>${this.escapeHtml(label)}</span>
      </div>
      <div class="timeline-section-items"></div>
    `;
    return div;
  },

  createTimelineItem(item) {
    const div = document.createElement("div");
    div.className = "timeline-item" + (item.is_deleted ? " deleted" : "");

    const type = this.detectType(item);
    const typeLabel = {
      activity: "动态归档",
      talk: "说说",
      blog: "日志",
      album: "相册",
      message: "留言",
      comment: "评论",
      visitor: "访客",
      video: "视频",
      like: "点赞",
      favorite: "收藏",
      diary: "日记",
      photo: "照片",
      mention: "提及",
      share: "转发",
      other: "动态",
    };
    const time = this.formatTime(
      item.publish_time ||
        item.message_time ||
        item.mention_time ||
        item.create_time ||
        item.created_at,
    );
    const content =
      item.content || item.title || item.name || item.summary || "";
    const authorQQ = item.user_qq || item.author_qq || this.qq;

    let imagesHTML = "";
    if (item.images) {
      try {
        const imgs =
          typeof item.images === "string"
            ? JSON.parse(item.images)
            : item.images;
        const filteredImgs = Array.isArray(imgs)
          ? imgs.filter((url) => !this.isAvatarLikeUrl(url))
          : [];
        if (filteredImgs.length > 0) {
          const allUrls = filteredImgs.map((url) => this.proxyImageUrl(url));
          const urlsJson = JSON.stringify(allUrls).replace(/'/g, "&#39;");
          const preview = this.buildDeferredImageTag(allUrls[0], {
            className: "deferred-image",
            onclick: `app.openLightbox(${urlsJson.replace(/"/g, "&quot;")}, 0)`,
          });
          const countBadge = filteredImgs.length > 1
            ? `<span class="timeline-image-count">共 ${filteredImgs.length} 张</span>`
            : "";
          imagesHTML = `<div class="timeline-images grid-1 timeline-images-compact">${preview}${countBadge}</div>`;
        }
      } catch {}
    }

    if (!imagesHTML && (item.cover || item.preview_url)) {
      const cover = item.cover || item.preview_url;
      if (!this.isAvatarLikeUrl(cover)) {
        const proxied = this.proxyImageUrl(cover);
        imagesHTML = `<div class="timeline-images grid-1">${this.buildDeferredImageTag(proxied, {
          className: "deferred-image",
        })}</div>`;
      }
    }

    // 相册：可点击的封面卡片
    if (type === "album" && (item.album_id || item.id)) {
      const albumId = item.album_id || item.id;
      const coverProxied = item.cover_url || item.cover ? this.proxyImageUrl(item.cover_url || item.cover) : "";
      const albumName = this.escapeHtml(item.name || item.title || "未命名相册");
      imagesHTML = `<div class="album-cover-card" onclick="app.openAlbumDetail('${albumId}', '${albumName.replace(/'/g, "\\'")}')">
        ${coverProxied ? this.buildDeferredImageTag(coverProxied, {
          className: "deferred-image",
        }) : ""}
        <div class="album-cover-info">
          <div class="album-name">${albumName}</div>
          <div class="album-count">点击查看相册内容</div>
        </div>
      </div>`;
    }

    const deletedBadge = item.is_deleted
      ? '<span class="deleted-badge">已删除</span>'
      : "";

    div.innerHTML = `
            <div class="timeline-header">
                ${this.buildDeferredImageTag(
                  this.proxyImageUrl(`https://q.qlogo.cn/headimg_dl?dst_uin=${authorQQ}&spec=100`),
                  { className: "timeline-avatar deferred-image" },
                )}
                <div class="timeline-meta">
                    <div class="timeline-author">${this.escapeHtml(item.author_name || this.nickname || this.qq)}</div>
                    <div class="timeline-time">${time}</div>
                </div>
                <span class="timeline-type type-${type}">${typeLabel[type] || "动态"}</span>
                ${deletedBadge}
            </div>
            <div class="timeline-body">${this.escapeHtml(content)}</div>
            ${imagesHTML}
            <div class="timeline-footer">
                ${item.like_count !== undefined ? `<span class="timeline-stat">点赞 ${item.like_count || 0}</span>` : ""}
                ${item.comment_count !== undefined ? `<span class="timeline-stat">评论 ${item.comment_count || 0}</span>` : ""}
                ${item.share_count !== undefined ? `<span class="timeline-stat">转发 ${item.share_count || 0}</span>` : ""}
                ${item.read_count !== undefined ? `<span class="timeline-stat">阅读 ${item.read_count || 0}</span>` : ""}
                ${item.photo_count !== undefined ? `<span class="timeline-stat">照片 ${item.photo_count || 0}</span>` : ""}
                ${item.source ? `<span class="timeline-stat">来源 ${this.escapeHtml(item.source)}</span>` : ""}
            </div>
        `;

    return div;
  },

  detectType(item) {
    if (item.type === "activity") return item.subtype || "activity";
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
          class="filter-chip ${this.friendGroupFilter === group.id ? "active" : ""}"
          onclick="app.switchFriendGroupFilter('${this.escapeHtml(group.id)}', event)"
        >${this.escapeHtml(group.name)}${group.id === "all" ? "" : ` · ${group.count}`}</button>
      `)
      .join("");

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
    return `<article class="friend-card">
      <div class="friend-card-head">
        ${this.buildDeferredImageTag(
          this.proxyImageUrl(friend.avatar || (`https://q.qlogo.cn/headimg_dl?dst_uin=${friend.friend_qq}&spec=100`)),
          { className: "friend-card-avatar deferred-image" },
        )}
        <div class="friend-card-meta">
          <h4>${this.escapeHtml(friend.name || friend.friend_qq)}</h4>
          <p>${this.escapeHtml(friend.remark || friend.friend_qq)}</p>
        </div>
        <span class="friend-card-badge">好友</span>
      </div>
      <div class="friend-card-body">
        <span>QQ：${this.escapeHtml(friend.friend_qq || "")}</span>
        ${friend.is_special_care ? "<span>特别关心</span>" : ""}
      </div>
    </article>`;
  },

  handleFriendSearch(event) {
    this.friendSearch = (event.target.value || "").trim();
    this.loadData();
  },

  switchFriendFilter(filter, event) {
    if (event) event.preventDefault();
    this.friendFilter = "current";
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
    document.getElementById("sidebar-progress").classList.remove("hidden");
    this.updateSyncStatus("同步中", "正在从 QQ 空间拉取并整理数据");
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
      document.getElementById("sidebar-progress-bar").style.width = pct + "%";
      document.getElementById("sidebar-progress-text").textContent =
        data.message ||
        `${data.current_type || "准备中"} (${data.done_types}/${data.total_types})`;
      this.applySyncStatus(data);

      if (data.status === "done" || data.status === "error") {
        this.stopSyncPoll();
        document.getElementById("sidebar-progress").classList.add("hidden");
        this.setSyncButtonState(false);

        if (data.status === "error") {
          this.updateSyncStatus("同步失败", data.error || "同步失败");
          if (this.handleAuthExpired(data.error)) return;
          alert(data.error || "同步失败");
        } else {
          this.updateSyncStatus("同步完成", "数据已更新，可以继续浏览当前内容");
        }

        // 刷新数据
        this.loadData();
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

  escapeHtml(str) {
    if (!str) return "";
    const div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  },

  proxyImageUrl(url) {
    if (!url) return "";
    if (url.includes(".qq.com") || url.includes(".qlogo.cn") || url.includes(".qpic.cn")) {
      return "/api/v1/proxy/image?url=" + encodeURIComponent(url);
    }
    return url;
  },

  isAvatarLikeUrl(url) {
    const value = String(url || "").trim().toLowerCase();
    if (!value) return false;
    return (
      value.includes("q.qlogo.cn/headimg_dl") ||
      value.includes("/headimg") ||
      value.includes("qlogo") ||
      value.includes("avatar")
    );
  },

  updateOverviewStats(total) {
    const el = document.getElementById("sidebar-total");
    if (el) el.textContent = total || 0;
  },

  updateSyncStatus(label, hint) {
    const syncEl = document.getElementById("sidebar-sync");
    const hintEl = document.getElementById("sidebar-sync-hint");
    if (syncEl) syncEl.textContent = label;
    if (hintEl) hintEl.textContent = hint || "";
  },

  applySyncStatus(data = {}) {
    const status = data.status || "idle";
    switch (status) {
      case "running":
        this.updateSyncStatus(
          "同步中",
          data.message || "正在从 QQ 空间拉取并整理数据",
        );
        break;
      case "done":
        this.updateSyncStatus(
          "同步完成",
          data.message || "数据已同步完成，可以继续浏览当前内容",
        );
        break;
      case "error":
        this.updateSyncStatus(
          "同步失败",
          data.error || data.message || "最近一次同步执行失败",
        );
        break;
      case "idle":
      default:
        this.updateSyncStatus(
          "未开始",
          "当前没有进行中的同步任务，可按需发起同步",
        );
        break;
    }
  },

  updateContentHead() {
    const meta = this.viewMeta[this.currentView] || this.viewMeta.memories;
    document.getElementById("content-kicker").textContent = meta.kicker;
    document.getElementById("content-title").textContent = meta.title;
    document.getElementById("content-description").textContent =
      meta.description;
  },

  formatTimelineGroup(timeStr) {
    if (!timeStr) return "更早以前";
    const d = new Date(timeStr);
    if (isNaN(d.getTime())) return "更早以前";
    return `${d.getFullYear()} 年 ${String(d.getMonth() + 1).padStart(2, "0")} 月`;
  },

  updateEmptyState() {
    const meta = this.viewMeta[this.currentView] || this.viewMeta.memories;
    document.getElementById("empty-title").textContent = meta.emptyTitle;
    document.getElementById("empty-description").textContent =
      meta.emptyDescription;
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
      const url = photo.url || photo.origin_url || "";
      if (!url) return;
      const proxied = this.proxyImageUrl(url);
      const item = document.createElement("div");
      item.className = "photo-item";
      item.innerHTML = this.buildDeferredImageTag(proxied, {
        className: "deferred-image",
        dataAttrs: {
          "album-photo": "true",
          fullsrc: proxied,
        },
        onclick: "app.openAlbumLightbox(this)",
      });
      if (photo.desc) {
        item.innerHTML += `<div class="photo-desc">${this.escapeHtml(photo.desc)}</div>`;
      }
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
      syncButton.textContent = loading ? "同步中" : "同步数据";
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

    return `<img src="${this.imagePlaceholder}" data-src="${safeSrc}" loading="lazy" decoding="async"${className}${alt}${onclick}${dataAttrs}>`;
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
    if (!target || this.currentView !== "memories") return;
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
