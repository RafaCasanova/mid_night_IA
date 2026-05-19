// Inicialização e configuração
(function () {
  'use strict';

  // Navegação lenta (já tratado por CSS, mas permitindo expansão futura)
  document.querySelectorAll('a.nav-link').forEach(link => {
    link.addEventListener('click', function (e) {
      e.preventDefault();
      const targetId = this.getAttribute('href');
      const targetEl = document.querySelector(targetId);
      if (targetEl) {
        targetEl.scrollIntoView({ behavior: 'smooth', block: 'start' });
        history.pushState(null, null, targetId);
      }
    });
  });

  // Ajuste visual para details expandíveis
  document.querySelectorAll('details').forEach(details => {
    details.addEventListener('toggle', () => {
      details.style.setProperty('--details-transition', details.open ? '0.2s' : '0.2s');
    });
  });

  // Garantir que a URL reflete a seção atual (na inicialização)
  if (window.location.hash) {
    const section = document.querySelector(window.location.hash);
    if (section) section.scrollIntoView({ behavior: 'instant', block: 'start' });
  }
})();
