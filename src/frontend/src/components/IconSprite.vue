<template>
  <svg class="sprite" aria-hidden="true" focusable="false">
    <symbol id="i-logo" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3.1" /><path d="M12 8.9V4.5M9.6 13.6 6 17.2M14.4 13.6l3.6 3.6" /><circle cx="12" cy="3.6" r="1.6" /><circle cx="5" cy="18.4" r="1.6" /><circle cx="19" cy="18.4" r="1.6" /></symbol>
    <symbol id="i-search" viewBox="0 0 24 24"><circle cx="11" cy="11" r="7" /><path d="m20 20-3.6-3.6" /></symbol>
    <symbol id="i-plus" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14" /></symbol>
    <symbol id="i-refresh" viewBox="0 0 24 24"><path d="M20 11a8 8 0 1 0-2.5 6.5" /><path d="M20 4v7h-7" /></symbol>
    <symbol id="i-folder" viewBox="0 0 24 24"><path d="M3 7.5A2.5 2.5 0 0 1 5.5 5H9l2 2.2h7.5A2.5 2.5 0 0 1 21 9.7v7.8a2.5 2.5 0 0 1-2.5 2.5h-13A2.5 2.5 0 0 1 3 17.5z" /></symbol>
    <symbol id="i-folder-open" viewBox="0 0 24 24"><path d="M3 8V6.5A2.5 2.5 0 0 1 5.5 4H9l2 2h7.5A2.5 2.5 0 0 1 21 8.5V10H3z" /><path d="M3 10h18l-2.2 8.2a2 2 0 0 1-1.9 1.5H5.6a2 2 0 0 1-1.9-1.5z" /></symbol>
    <symbol id="i-settings" viewBox="0 0 24 24"><path d="M4 6h8M18 6h2M4 12h2M10 12h10M4 18h8M18 18h2" /><circle cx="15" cy="6" r="2.2" /><circle cx="7" cy="12" r="2.2" /><circle cx="15" cy="18" r="2.2" /></symbol>
    <symbol id="i-grid" viewBox="0 0 24 24"><rect x="3.5" y="3.5" width="6.5" height="6.5" rx="1" /><rect x="14" y="3.5" width="6.5" height="6.5" rx="1" /><rect x="3.5" y="14" width="6.5" height="6.5" rx="1" /><rect x="14" y="14" width="6.5" height="6.5" rx="1" /></symbol>
    <symbol id="i-list" viewBox="0 0 24 24"><path d="M8 6h13M8 12h13M8 18h13M3.5 6h.01M3.5 12h.01M3.5 18h.01" /></symbol>
    <symbol id="i-x" viewBox="0 0 24 24"><path d="M18 6 6 18M6 6l12 12" /></symbol>
    <symbol id="i-check" viewBox="0 0 24 24"><path d="m20 6-11 11-5-5" /></symbol>
    <symbol id="i-link" viewBox="0 0 24 24"><path d="M10 13a5 5 0 0 0 7.5.5l2-2a5 5 0 0 0-7-7l-1.1 1.1" /><path d="M14 11a5 5 0 0 0-7.5-.5l-2 2a5 5 0 0 0 7 7l1.1-1.1" /></symbol>
    <symbol id="i-copy" viewBox="0 0 24 24"><rect x="9" y="9" width="12" height="12" rx="2" /><path d="M5 15V5a2 2 0 0 1 2-2h10" /></symbol>
    <symbol id="i-external" viewBox="0 0 24 24"><path d="M15 3h6v6M10 14 21 3" /><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" /></symbol>
    <symbol id="i-chevron-down" viewBox="0 0 24 24"><path d="m6 9 6 6 6-6" /></symbol>
    <symbol id="i-chevron-right" viewBox="0 0 24 24"><path d="m9 18 6-6-6-6" /></symbol>
    <symbol id="i-arrow-down" viewBox="0 0 24 24"><path d="M12 4v15M6.5 13.5 12 19l5.5-5.5" /></symbol>
    <symbol id="i-arrow-up-right" viewBox="0 0 24 24"><path d="M7 17 17 7M8 7h9v9" /></symbol>
    <symbol id="i-more" viewBox="0 0 24 24"><circle cx="5" cy="12" r="1" /><circle cx="12" cy="12" r="1" /><circle cx="19" cy="12" r="1" /></symbol>
    <symbol id="i-filter" viewBox="0 0 24 24"><path d="M4 5h16l-6.2 7.2V19L10.2 21v-8.8z" /></symbol>
    <symbol id="i-book" viewBox="0 0 24 24"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" /><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" /></symbol>
    <symbol id="i-share" viewBox="0 0 24 24"><circle cx="6" cy="12" r="2.4" /><circle cx="18" cy="6" r="2.4" /><circle cx="18" cy="18" r="2.4" /><path d="m8.2 10.9 7.6-3.7M8.2 13.1l7.6 3.7" /></symbol>
    <symbol id="i-database" viewBox="0 0 24 24"><ellipse cx="12" cy="5" rx="7.5" ry="2.8" /><path d="M4.5 5v6.5c0 1.5 3.4 2.8 7.5 2.8s7.5-1.3 7.5-2.8V5" /><path d="M4.5 11.5V18c0 1.5 3.4 2.8 7.5 2.8s7.5-1.3 7.5-2.8v-6.5" /></symbol>
    <symbol id="i-sun" viewBox="0 0 24 24"><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></symbol>
    <symbol id="i-moon" viewBox="0 0 24 24"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" /></symbol>
    <symbol id="i-palette" viewBox="0 0 24 24"><path d="M12 3a9 9 0 0 0 0 18h1.2a2.1 2.1 0 0 0 1.4-3.7 1.7 1.7 0 0 1 1.1-3h1.9A4.4 4.4 0 0 0 22 9.9C22 6.1 18.4 3 12 3z" /><circle cx="7.5" cy="10" r="1" /><circle cx="10.5" cy="6.8" r="1" /><circle cx="15" cy="7.2" r="1" /><circle cx="17.8" cy="11" r="1" /></symbol>
    <symbol id="i-minus" viewBox="0 0 24 24"><path d="M5 12h14" /></symbol>
    <symbol id="i-maximize" viewBox="0 0 24 24"><rect x="5" y="5" width="14" height="14" rx="1" /></symbol>
    <symbol id="i-shield" viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /><path d="m9 12 2 2 4-4" /></symbol>
    <symbol id="i-alert" viewBox="0 0 24 24"><path d="M12 9v4M12 17h.01" /><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" /></symbol>
    <symbol id="i-clock" viewBox="0 0 24 24"><circle cx="12" cy="12" r="8.5" /><path d="M12 7v5.2l3.2 2" /></symbol>
    <symbol id="i-download" viewBox="0 0 24 24"><path d="M12 3v12M7 10l5 5 5-5" /><path d="M4.5 19.5h15" /></symbol>
    <symbol id="i-trash" viewBox="0 0 24 24"><path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6" /></symbol>
    <symbol id="i-edit" viewBox="0 0 24 24"><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" /></symbol>
    <symbol id="i-file" viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><path d="M14 2v6h6" /></symbol>
    <symbol id="i-git" viewBox="0 0 24 24"><circle cx="6" cy="6" r="2.3" /><circle cx="6" cy="18" r="2.3" /><circle cx="18" cy="6" r="2.3" /><path d="M6 8.3v7.4M8.3 6h5A2.3 2.3 0 0 1 15.7 8.3v3" /></symbol>
    <symbol id="i-package" viewBox="0 0 24 24"><path d="m12 2 9 5v10l-9 5-9-5V7z" /><path d="m3 7 9 5 9-5M12 12v10" /></symbol>
    <symbol id="i-inbox" viewBox="0 0 24 24"><path d="M3 12h5l2 3h4l2-3h5" /><path d="M5 5h14l2 7v7H3v-7z" /></symbol>
    <symbol id="i-sparkle" viewBox="0 0 24 24"><path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" /></symbol>
    <symbol id="i-check-circle" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" /><path d="m8.5 12 2.4 2.4 4.6-5" /></symbol>
    <symbol id="i-terminal" viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2" /><path d="m7 9 3 3-3 3M13 15h4" /></symbol>
    <symbol id="i-info" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9" /><path d="M12 11v5M12 8h.01" /></symbol>
    <symbol id="i-sliders" viewBox="0 0 24 24"><path d="M4 6h10M18 6h2M4 12h4M12 12h8M4 18h10M18 18h2" /><circle cx="16" cy="6" r="2" /><circle cx="8" cy="12" r="2" /><circle cx="16" cy="18" r="2" /></symbol>
    <symbol id="i-monitor" viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="13" rx="2" /><path d="M8 21h8M12 17v4" /></symbol>
  </svg>
</template>
