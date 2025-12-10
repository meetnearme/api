var DIGIOH_LOADER = DIGIOH_LOADER || {};
(function (digioh_loader) {
    if (digioh_loader.loaded) { return; }
    digioh_loader.loaded = true;

    var isMainLoaded = false;

    function loadMain() {
        if (!isMainLoaded) {
            isMainLoaded = true;

            var e = document.createElement('script'); e.type = 'text/javascript'; e.async = true;
            e.src = '//forms.6amcity.com/w37htfhcq2/vendor/0d398f98-f30e-4cb9-9ca0-2e327cf1d66b/user' + ((window.sessionStorage.getItem('xdibx_boxqamode') == 1 || window.location.href.indexOf('boxqamode') > 0)  ? '_qa' : '') + '.js?cb=638980661542763054';
            var s = document.getElementsByTagName('script')[0]; s.parentNode.insertBefore(e, s);
        }
    };

    function sendPV() {
        try {
            window.SENT_LIGHTBOX_PV = true;

            var hn = 'empty';
            if (window && window.location && window.location.hostname) {
                hn = window.location.hostname;
            }

            var i = document.createElement("img");
            i.width = 1;
            i.height = 1;
            i.src = ('https://forms.6amcity.com/w37htfhcq2/z9g/digibox.gif?c=' + (new Date().getTime()) + '&h=' + encodeURIComponent(hn) + '&e=p&u=45268');
        }
        catch (e) {
        }
    };

    sendPV();
    loadMain();
})(DIGIOH_LOADER);