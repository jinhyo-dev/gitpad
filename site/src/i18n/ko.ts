import type { Dict } from "./en";

export const ko: Dict = {
  locale: "ko",
  meta: {
    title: "gitpad · 터미널 Git TUI — 커밋 그래프·브랜치·푸시",
    description:
      "gitpad는 오픈소스 Git TUI입니다. 커밋 그래프, 브랜치 트리, 디프, 커밋 워크스페이스, Push 다이얼로그, GitHub CI 상태를 터미널 한 화면에서. macOS · Linux · Windows 단일 바이너리.",
    keywords: [
      "gitpad",
      "git tui",
      "git 터미널",
      "github 터미널",
      "터미널 git 클라이언트",
      "git 로그 그래프",
      "git 브랜치 관리",
      "git cli ui",
      "terminal ui",
      "TUI",
      "git",
      "github",
      "lazygit 대안",
      "macos",
      "linux",
      "windows",
    ],
    siteName: "gitpad",
    ogLocale: "ko_KR",
  },
  nav: {
    about: "소개",
    features: "기능",
    demo: "데모",
    install: "설치",
    keys: "단축키",
    github: "GitHub",
  },
  hero: {
    eyebrow: "오픈소스 · Go · 단일 바이너리",
    title: "Git 히스토리 전체를\n한 화면에.",
    subtitle:
      "gitpad는 터미널에서 동작하는 Git UI입니다. 브랜치 트리, 컬러 커밋 그래프, 변경 파일과 커밋 상세를 나란히 보여주고, 체크아웃·머지·리베이스·체리픽·리셋은 컨텍스트 메뉴로, 커밋과 푸시는 전용 화면에서 처리합니다.",
    tryDemo: "데모 해보기",
    install: "설치하기",
    copy: "복사",
    copied: "복사됨!",
  },
  demo: {
    title: "지금 바로 써보기",
    subtitle:
      "브라우저에서 동작하는 gitpad 재현본입니다. 가상의 저장소 위에서 실제와 같은 키 조작이 됩니다 — 클릭한 뒤 키보드를 써보세요.",
    focusTitle: "클릭해서 시작",
    focusHint: "그 다음은 키보드로 — 방향키, enter, /, c, P…",
    focusOn: "키보드 입력 중",
    reset: "데모 초기화",
    keys: [
      ["↑ ↓ / j k", "이동"],
      ["1 2 3 / tab", "패널 전환"],
      ["enter", "메뉴 / 열기"],
      ["/", "검색"],
      ["c", "커밋"],
      ["P", "푸시"],
      ["esc", "뒤로"],
    ],
  },
  about: {
    title: "로그를 중심으로 만든 Git TUI",
    paragraphs: [
      "gitpad는 터미널용 Git 사용자 인터페이스(TUI)입니다. git log, git branch, git diff를 번갈아 치는 대신 커밋 그래프, 브랜치 트리, 변경 파일, 커밋 상세를 한 화면에서 봅니다 — 데스크톱 Git 클라이언트의 화면 구성을, 이미 쓰고 있는 터미널 안에서.",
      "Go로 만든 단일 바이너리이며 여러분의 git을 그대로 호출하므로 훅, credential helper, 서명 키가 그대로 동작합니다. GitHub에 올라간 저장소는 커밋마다 체크 상태가 표시되고, 커밋 워크스페이스와 Push 다이얼로그가 키보드에서 손을 떼지 않고 GitHub 워크플로를 처리합니다.",
    ],
    facts: [
      ["플랫폼", "macOS · Linux · Windows"],
      ["설치", "Homebrew · apt · Scoop · go install"],
      ["라이선스", "MIT, 오픈소스"],
      ["언어", "Go (bubbletea)"],
    ],
  },
  features: {
    title: "로그 뷰에 있어야 할 모든 것",
    subtitle: "데스크톱 Git 클라이언트의 경험을, 터미널을 떠나지 않고.",
    items: [
      {
        title: "커밋 그래프",
        body: "커밋마다 레인·머지·브랜치 헤드를 그리고 HEAD, 브랜치, 리모트, 태그를 칩으로 표시합니다.",
      },
      {
        title: "컨텍스트 메뉴",
        body: "커밋이나 브랜치에서 우클릭 또는 Enter: 체크아웃, 머지, 리베이스, 체리픽, 리버트, 리셋, 새 브랜치·태그.",
      },
      {
        title: "커밋 워크스페이스",
        body: "원하는 파일만 체크하고, 히스토리 불러오기가 되는 여러 줄 메시지를 쓰고, 파일별 디프를 보면서 커밋 또는 커밋 & 푸시.",
      },
      {
        title: "Push 다이얼로그",
        body: "어떤 커밋이 올라가는지 미리 보고, 브랜치가 뒤처졌으면 경고를 받고, force-with-lease나 태그 푸시를 토글.",
      },
      {
        title: "CI 상태",
        body: "GitHub 체크 결과를 커밋 옆에 ✓ ✗ ◌ 로 표시하고, 상세 패널에서 개별 실행과 소요 시간을 확인.",
      },
      {
        title: "검색과 필터",
        body: "메시지·작성자·해시를 한 검색창으로, 타이핑 필터가 되는 브랜치 선택기, 파일 히스토리 필터.",
      },
    ],
  },
  install: {
    title: "한 줄로 설치",
    subtitle:
      "macOS, Linux, Windows용 빌드 제공. gitpad는 여러분의 git을 그대로 호출하므로 훅, credential helper, 서명이 그대로 동작합니다.",
    tabs: { mac: "macOS", debian: "Debian / Ubuntu", windows: "Windows", go: "Go" },
    note: "설치 후 아무 저장소 안에서 gitpad 를 실행하세요.",
    releases: "전체 릴리즈",
  },
  keys: {
    title: "키보드 우선",
    subtitle: "모든 동작은 키 하나. 마우스도 됩니다.",
    groups: [
      {
        title: "이동",
        rows: [
          ["tab · 1 2 3", "패널 전환"],
          ["j k ↑ ↓", "이동"],
          ["g / G", "처음 / 끝"],
          ["← →", "접기/펼치기, 그 다음 이전/다음 패널"],
          ["맨 위에서 ↑", "검색창으로"],
        ],
      },
      {
        title: "로그",
        rows: [
          ["enter / m / 우클릭", "커밋 액션"],
          ["/", "메시지·작성자·해시 검색"],
          ["A", "전체 브랜치 ↔ 현재 브랜치"],
          ["y", "해시 복사"],
          ["v", "새 버전 태그 (patch / minor / major)"],
        ],
      },
      {
        title: "커밋 & 푸시",
        rows: [
          ["c / C", "커밋 워크스페이스"],
          ["enter", "파일 체크 / 해제"],
          ["⌃S · ⌃P", "커밋 · 커밋 & 푸시"],
          ["P", "Push 다이얼로그"],
          ["p", "Pull (merge / rebase / fetch)"],
        ],
      },
      {
        title: "디프",
        rows: [
          ["enter", "디프 열기"],
          ["↑ ↓", "다음 / 이전 변경 블록"],
          ["n / p", "다음 / 이전 파일"],
          ["esc", "뒤로"],
        ],
      },
    ],
  },
  footer: {
    license: "MIT 라이선스",
    source: "GitHub 소스",
    lang: "Language: English",
  },
};
