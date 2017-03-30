
var AppConfig = {guest: false, wideview: false};

var m = angular.module('souris-common', [ 'ngRoute', 'ngSanitize', 'ngCookies', 'ui.bootstrap']);
m.config(['$locationProvider', function(l){l.html5Mode(true);}]);

m.directive('sourisSiteNav', function () {
    return {
        scope: {
            cfg: '='
        },
        controller: ['$scope', '$location', '$remoteService',
        function ($scope, $location, req) {
            var dom = $location.host();
            var url = $location.absUrl().split('#')[0].replace(/\/$/, '');

            var base = url.split(dom +'/',2);

            if(base.length == 1) base = '/';
            else base = base[1].split('/',2)[0];

            $scope.cfg = {
                title: 'Souris Framework',
                domain: dom,
                url: url,
                abs_url: $location.absUrl(),
                base: base,
                wideview: AppConfig.wideview,
                guest: AppConfig.guest,
                denied: false,
                app_nav: []
            };

            function extend(obj, props) {
                for(var prop in props) {
                    if(props.hasOwnProperty(prop)) {
                        obj[prop] = props[prop];
                    }
                }
            }

            req('cfg.json')
                .get({dom: dom, base:base=='/'?'root':base})
                .success(function(d){extend($scope.cfg,d);})
                .error(function(){});

            req('/app-info')
                .get()
                .success(function(d){$scope.cfg.version = d;})
        }],
        templateUrl: 'souris-site-nav-template',
        replace: true
    };
});

m.directive('sourisAppNav', function () {
    return {
        scope: {
            cfg: '='
        },
        controller: function () {},
        templateUrl: 'souris-app-nav-template',
        replace: true
    };
});

m.directive('sourisSidemenu', function () {
    return {
        scope: false,
        controller: ['$scope',function ($scope) {
            if ($scope.$env == undefined) $scope.$root.$env = {};
        }],
        templateUrl: 'souris-sidemenu-template',
        replace: false
    };
});

m.factory('$remoteService', ['$http','$cookies',
  function($http){
    var execute = function (fn, method, url, query, payload) {
        //var config = {cache: false, headers: { 'If-Modified-Since': "0" }, withCredentials: true};
        if (query === undefined) query = {};
        for (var i in query) if (query.hasOwnProperty(i)) {
            var needle = ':'+i;
            if (url.search(needle)>0) {
                url = url.replace(needle, query[i]);
                delete(query[i]);
            }
        }

        var config = {};
        config.method = method;
        config.params = query;
        config.url = url;

        if (payload !== undefined) config.data = payload;
        if (fn !== undefined) config = fn(config);

        return $http(config);
    };

    return function(url, fn) {
        return {
            get:      function (query)          { return execute(fn, 'GET',    url, query); },
            put:      function (query, payload) { return execute(fn, 'PUT',    url, query, payload); },
            post:     function (query, payload) { return execute(fn, 'POST',   url, query, payload); },
            patch:    function (query, payload) { return execute(fn, 'PATCH',  url, query, payload); },
            del:      function (query)          { return execute(fn, 'DELETE', url, query); },
            'delete': function (query)          { return execute(fn, 'DELETE', url, query); },
            'remove': function (query)          { return execute(fn, 'DELETE', url, query); },
            jsonp:    function (query)          { return execute(fn, 'JSONP',  url, query); }
        };

    };
}]);
m.factory('$rubiconRemoteService', ['$remoteService','$cookies',
    function(req, cookies){
        return function(url) {
            var fn = function(config) {
                if (config.params === undefined) config.params = {};
                if (cookies.get('user_token') !== undefined)
                    config.params['user_token'] = cookies.get('user_token');

                return config;
            };

            return req(url, fn);
        };
    }
]);
m.factory('$sourisRemoteService', ['$remoteService','$cookies',
    function(req, cookies){
        return function(url) {
            var fn = function(config) {
                if (config.headers === undefined) config.headers = {};
                if (cookies.get('souris_token') !== undefined)
                    config.headers['authorization'] = "souris " + cookies.get('souris_token');
                return config;
            };
            return req(url, fn);
        };
    }
]);

m.filter('default', function() { return function (a, b) { return a==undefined||a==""?b:a; }; });
m.filter('lpad', function () {
    return function (v,l,p) {
        if (v === undefined || l === undefined) return "";
        if (p === undefined) p = '0';
        return (new Array(l+1).join(p) + v).slice(-l);
    };
});
m.filter('startFrom', function () {
    return function (input, start) {
        if (input === undefined) return [];
        if (input.slice === undefined) return [];
        start = +start; //parse to int
        return input.slice(start);
    };
});
m.filter('startsWith', function () {
    return function (input, fields) {
        if (input === undefined) return [];
        if (input.slice === undefined) return [];
        if (fields === undefined) return input;

        return input.filter(function(s) {
            for (var i in fields) if (fields.hasOwnProperty(i)) {
                if (s[i]!==undefined)
                    return s[i].substr(0, fields[i].length) == fields[i];
            }
            return false;
        });
    };
});
m.filter('beforeChar', function () {
    return function (input, char) {
        if (input == undefined) return "";
        if (char == undefined) char = ' ';
        var index = input.indexOf(char);
        if (index != -1) {
            return input.substring(0, index);
        }
        return input;
    };
});

m.filter('sourisPagerFilter', ['$filter', function ($filter) {
    return function (input, page) {
        if (page === undefined) page = {};
        return $filter('limitTo')(
            $filter('startFrom')(
                $filter('filter')(input, page.search),
                page.current * page.size),
            page.size);
    };
}]);
m.directive('sourisPager', ['$filter', function ($filter) {
    return {
        restrict: 'A',
        scope: {
            pagerPage: '=',
            pagerItems: '=',
            pagerSize: '=',
            pagerSearch: '='
        },
        controller: ['$scope', function ($scope) {
            var search = {},
                size = 20;

            if ($scope.pagerPage !== undefined && $scope.pagerPage.search !== undefined)
                search = $scope.pagerPage.search;

            if ($scope.pagerSize !== undefined)
                size = $scope.pagerSize;
            if ($scope.pagerSearch !== undefined)
                search = $scope.pagerSearch;

            $scope.pagerPage = {
                search: search,
                current: 0,
                last: 0,
                size: size,
                count: function () {
                    var d = $scope.pagerItems;
                    if (d === undefined) return 1;
                    var i = Math.ceil($filter('filter')(d, this.search).length / this.size);

                    this.last = i;
                    if (this.current > i - 1) this.current = 0;

                    return i;
                }
            };
            $scope.prev = function(){if ($scope.pagerPage.current > 0) $scope.pagerPage.current = $scope.pagerPage.current - 1; };
            $scope.next = function(){if ($scope.pagerPage.current < $scope.pagerPage.last - 1) $scope.pagerPage.current = $scope.pagerPage.current + 1; };

            $scope.pagerPage.count();

        }],
        template: "<nav><ul class=pager>" +
        "<li class=previous ng-class='{\"disabled\":pagerPage.current == 0}'><a ng-click='prev()'><span aria-hidden='true'>&larr;</span> Previous</a></li>" +
        "{{pagerPage.current+1}}/{{pagerPage.count()}}" +
        "<li class='next' ng-class='{\"disabled\": pagerPage.current >= pagerPage.last - 1}'><a ng-click='next()'>Next <span aria-hidden='true'>&rarr;</span></a></a></li>" +
        "</ul></nav>",
        replace: false
    };
}]);
m.directive('sourisSorter', function () {
    return {
        restrict: 'A',
        scope: {
            sorterOrder: '=',
            sorterItems: '='
        },
        controller: ['$scope', function ($scope) {
            $scope.sort = function (s) {
                var n = '+' + s;
                if (n == $scope.sorterOrder) n = '-' + s;
                $scope.sorterOrder = n;
            };
        }],
        template: "<th ng-repeat='i in sorterItems' ng-click='sort(i[0])' style=\"width: {{i[2]|default:'auto'}}\"> {{i[1]}} &nbsp;" +
        "<i class='glyphicon glyphicon-chevron-up' ng-show='sorterOrder==\"+\" + i[0]'></i>" +
        "<i class='glyphicon glyphicon-chevron-down' ng-show='sorterOrder==\"-\" + i[0]'></i>" +
        "</th>"
    };
});
m.directive('sourisUserMock',function(){
    return {
        restrict: 'A',
        scope: {
            user: '='
        },
        controller: ['$scope','$remoteService','$uibModal', '$timeout', '$cookies', '$route',
            function ($scope, req, $modal, $timeout, $cookies, $route) {

                $scope.user = {};
                var user = $scope.user;

                user.ident = 'anon';
                user.active = false;
                user.token = $cookies.get('user_token');
                user.roles = [];
                user.groups = [];
                user.$role = function (role) {
                    if (role instanceof Array) {
                        for (var i in role) {
                            if (!role.hasOwnProperty(i)) continue;
                            if (user.roles.indexOf(role[i]) > -1) return true;
                        }
                        return false;
                    }
                    return (user.roles.indexOf(role) > -1);
                };
                user.$group = function (group) {
                    if (group instanceof Array) {
                        for (var i in group) {
                            if (!group.hasOwnProperty(i)) continue;
                            if (user.groups.indexOf(group[i]) > -1) return true;
                        }
                        return false;
                    }
                    return (user.groups.indexOf(group) > -1);
                };

                function refresh() {
                    req('/user.json')
                        .get()
                        .success(setUser)
                        .error(setUser);
                }
                refresh();

                function setUser(u) {
                    var reflow = false;
                    if (user.ident != u.userName) reflow = true;

                    user.ident = u.userName;
                    user.display_name = u.firstName + " " + u.lastName;
                    user.email = u.email;
                    user.active = u.loggedIn;
                    user.groups = u.groups===undefined ? [] : u.groups;
                    user.roles = u.roles===undefined ? [] : u.roles;

                    if (user.active)
                        $timeout(refresh, 72000);

                    if (reflow) {
                        console.log("Reflow Occurred.");
                        $route.reload();
                    }
                }

            }],
        templateUrl: 'souris-user-mock'
    };
});
m.directive('sourisUserEd25519',function(){
    return {
        restrict: 'A',
        scope: {
            user: '='
        },
        controller: ['$scope','$remoteService','$uibModal', '$timeout', '$route',
            function ($scope, req, $modal, $timeout, $route) {

                $scope.user = {};
                var user = $scope.user;

                user.ident = '@anon';
                user.secret = '';
                user.pubkey = '';
                user.active = false;

                if (typeof(Storage) !== "undefined") {
                    if (localStorage.getItem("active")) {
                        setUser({
                            ident: localStorage.getItem("ident"),
                            secret: localStorage.getItem("secret"),
                            active: true
                        });
                    }
                }
                function decSK(secretKey) {
                    try {
                        var k = (secretKey instanceof Uint8Array?secretKey:nacl.util.decodeBase64(secretKey));
                        if (k.length != nacl.sign.secretKeyLength) {
                            self.result = {err:true, msg:'Bad secret key length: must be ' + nacl.sign.secretKeyLength + ' bytes'};
                            return null;
                        }
                        return k;
                    } catch(e) {
                        self.result = {err:true, msg:'Failed to decode secret key from Base64'};
                        return null;
                    }
                }
                function getPK(sk) {
                    sk = decSK(sk);
                    if (sk === null) return null;
                    var keys = nacl.sign.keyPair.fromSecretKey(sk);
                    return nacl.util.encodeBase64(keys.publicKey);
                }

                function setUser(u) {
                    var reflow = false;
                    if (user.ident != u.ident) reflow = true;

                    user.ident = u.ident;
                    user.secret = u.secret;
                    user.pubkey = getPK(u.secret);
                    user.active = u.active;

                    if (reflow) {
                        console.log("Reflow Occurred.");
                        $route.reload();
                    }
                }

                user.$login = function(u, p) {
                    if (typeof(Storage) !== "undefined") {
                        localStorage.setItem("ident",  u);
                        localStorage.setItem("secret", p);
                        localStorage.setItem("active", true);
                    }
                    setUser({ident: u, secret: p, active: true});
                };
                user.$logout = function(){
                    if (typeof(Storage) !== "undefined") {
                        localStorage.setItem("ident",  'anon');
                        localStorage.setItem("secret", '');
                        localStorage.setItem("active", false);
                    }
                    setUser({ident:"anon",secret:'',active:false});
                    $route.reload();
                };

                $scope.openLogin = function(){
                    $modal.open({
                        animation: true,
                        size: 'sm',
                        templateUrl: 'souris-user-ed25519-login',
                        controller: ['$scope',
                            function ($scope) {
                                $scope.user = "";
                                $scope.secret = "";
                                $scope.pubkey = '';
                                $scope.$login = function(user, secret) {
                                    $scope.$dismiss();
                                    $scope.$user.$login(user, secret);
                                };
                            }]
                    })
                };
            }],
        templateUrl: 'souris-user-ed25519'
    };
});

m.directive('sourisUserPasswd',function(){
    return {
        restrict: 'A',
        scope: {
            user: '='
        },
        controller: ['$scope','$sourisRemoteService','$uibModal', '$timeout', '$cookies', '$route',
            function ($scope, req, $modal, $timeout, $cookies, $route) {

                $scope.user = {};
                var user = $scope.user;

                user.ident = 'anon';
                user.active = false;
                user.token = $cookies.get('user_token');
                user.roles = [];
                user.groups = [];
                user.error = false;

                user.$role = function (role) {
                    if (role instanceof Array) {
                        for (var i in role) {
                            if (!role.hasOwnProperty(i)) continue;
                            if (user.roles.indexOf(role[i]) > -1) return true;
                        }
                        return false;
                    }
                    return (user.roles.indexOf(role) > -1);
                };

                user.$group = function (group) {
                    if (group instanceof Array) {
                        for (var i in group) {
                            if (!group.hasOwnProperty(i)) continue;
                            if (user.groups.indexOf(group[i]) > -1) return true;
                        }
                        return false;
                    }
                    return (user.groups.indexOf(group) > -1);
                };

                function refresh() {
                    req('/profile/user.profile')
                        .get()
                        .success(setUser)
                        .error(setUser);
                }
                refresh();

                function setUser(u) {
                    var reflow = false;
                    if (user.ident != u.ident) reflow = true;

                    if (u.token !== undefined)
                        $cookies.put("souris_token", u.token, {path: "/"});

                    user.ident = u.ident;
                    user.aspect = u.aspect;
                    user.site = u.site;
                    user.local = u.local;
                    user.app = u.app;

                    user.display_name = u.ident;

                    if (u.site.first_name !== undefined && u.site.last_name != undefined)
                        user.display_name = u.site.first_name + ' ' + u.site.last_name;

                    if (u.site.display_name !== undefined)
                        user.display_name = u.site.display_name;

                    user.email = u.site.email;

                    user.active = u.ident != 'anon';
                    user.token = $cookies.get('souris_token');
                    user.groups = u.groups===undefined ? [] : u.groups;
                    user.roles = u.roles===undefined ? [] : u.roles;
                    user.error = false;

                    if (user.active)
                        $timeout(refresh, 72000);

                    if (reflow) {
                        console.log("Reflow Occurred.");
                        $route.reload();
                    }
                }

                user.$login = function(u, p) {
                    req('/profile/user.session')
                        .post({},{ident: u, password: p, aspect: '*'})
                        .success(setUser)
                        .error(function(){
                            user.error = true;
                        });
                };

                user.$logout = function(){
                    req('/profile/user.session').delete();

                    setUser({ident:"anon", aspect:"default", site:{display_name:"Guest User", email:"nobody@nowhere"}});
                    $cookies.remove("souris_token");
                    $route.reload();
                };

                $scope.openLogin = function(){
                    $modal.open({
                        animation: true,
                        size: 'sm',
                        templateUrl: 'souris-user-passwd-login',
                        controller: ['$scope',
                            function ($scope) {
                                $scope.user = "";
                                $scope.pass = "";
                                $scope.token = "";
                                $scope.$login = function(user, pass) {
                                    $scope.$dismiss();
                                    $scope.$user.$login(user, pass);
                                };
                                $scope.$token = function(token) {
                                    $scope.$dismiss();
                                    $scope.$user.$token(token);
                                };
                            }]
                    })
                };
            }],
        templateUrl: 'souris-user-passwd'
    };
});

m.filter('markdown', ['$sce', '$anchorScroll', function($sce, $anchorScroll){
    var md = new Remarkable({xhtmlOut:true, breaks:true, linkify:true});

    var wiki_links = function(opts){
        var pfx = '';
        if (opts.pfx !== undefined) pfx = opts.pfx;

        return function(state, silent) {
            var end = -1,
                mid = -1,
                name,
                pos = state.pos,
                code,
                oldPos = state.pos,
                max = state.posMax,
                start = state.pos,
                marker = state.src.charCodeAt(start),
                marker2 = state.src.charCodeAt(start+1);

            if (state.src.charCodeAt(pos) !== 0x5B/* [ */) { return false; }
            if (state.src.charCodeAt(pos+1) !== 0x5B/* [ */) { return false; }
            start = pos+2;
            for (pos=start; pos < max; pos++) {
                if (state.src.charCodeAt(pos) === 0x5D/* ] */ &&
                    state.src.charCodeAt(pos+1) === 0x5D/* ] */) {
                    end = pos;
                    break;
                } else if (state.src.charCodeAt(pos) === 0x7C/* | */) {
                    mid = pos;
                }
            }

            // parser failed to find ']', so it's not a valid link
            if (end < 0) { return false; }

            if (pos >= max || state.src.charCodeAt(pos) !== 0x5D/* ] */) {
                state.pos = oldPos;
                return false;
            }

            if (mid > 0) {
                name = state.src.slice(start, mid);
                href = pfx + state.src.slice(mid+1, end);
            } else {
                name = state.src.slice(start, end);
                href = pfx + name.toLowerCase().replace(/\s+/g, '-');
            }

            //
            // We found the end of the link, and know for a fact it's a valid link;
            // so all that's left to do is to call tokenizer.
            //
            if (!silent) {
                state.push({
                    type: 'link_open',
                    href: href,
                    title: name,
                    level: state.level++
                });
                state.pending = name;
                state.push({ type: 'link_close', level: --state.level });
            }

            state.pos = pos + 2;
            state.posMax = max;
            return true;
        };
    }
    var table_open = function(opts) {
	    return function() { var cls= opts.class === undefined?'':' class="'+opts.class+'"'; return '<table'+cls+'>\n'; };
    };
    var heading_open = function() {
        return function(tokens, idx) {
            var text = tokens[idx+1].content === undefined
                     ? ''
                     : tokens[idx+1].content
                                    .toLowerCase()
                                    .replace(/\s+/g, '-')
                                    .replace(/[^a-z0-9\-]+/gi, '');
            var id = text == '' ? '' : ' id="' + text + '"';
            return '<h' + tokens[idx].hLevel + id + '>';
        };
    };

    md.use(function(md, opts){
 	md.inline.ruler.before('links', 'wiki_links', wiki_links(opts), opts);
	md.renderer.rules.table_open = table_open(opts);
        md.renderer.rules.heading_open = heading_open();
    }, {class:'table table-striped', pfx:'/wiki/'});

    return function(s){ s=""+s; if (s==='') return ''; $anchorScroll(); return $sce.trustAsHtml(md.render(s)); };
}]);

var app = angular.module('souris-app', ['souris-common']).
    config(['$routeProvider', function($routeProvider) {
    $routeProvider.
    when('/',        { controller: function() {}, templateUrl: 'default/failtoload.tmpl' }).
    otherwise({redirectTo:'/'});
}]).
    run(['$templateCache', function($templateCache) {
    $templateCache.put('default/failtoload.tmpl',
        "<h1>Failed to Load!</h1>" +
        "<p>The Application you are trying to access was not found.</p>" +
        "<p>Perhaps you are in the wrong location?</p>" +
        "<p><a href='/'>Click here to return home.</a></p>" +
        "<h4>Debug Info</h4>" +
        "<pre>$cfg = {{$cfg|json}}\n\n$user = {{$user|json}}</pre>");
}]);

!function(e, n) {
    "use strict";
    "undefined" != typeof module && module.exports
        ? module.exports = n()
        : e.nacl
        ? e.nacl.util = n()
        : (e.nacl = {}, e.nacl.util = n())
}(this, function() {
    "use strict";
    var e = {};
    return e.decodeUTF8 = function(e) {
        var n,
            t = decodeURIComponent(encodeURIComponent(e)),
            r = new Uint8Array(t.length);
        for (n = 0; n < t.length; n++)
            r[n] = t.charCodeAt(n);
        return r
    },
        e.encodeUTF8 = function(e) {
            var n,
                t = [];
            for (n = 0; n < e.length; n++)
                t.push(String.fromCharCode(e[n]));
            return decodeURIComponent(encodeURIComponent(t.join("")))
        },
        e.encodeBase64 = function(e) {
            if ("undefined" == typeof btoa)
                return new Buffer(e).toString("base64");
            var n,
                t = [],
                r = e.length;
            for (n = 0; r > n; n++)
                t.push(String.fromCharCode(e[n]));
            return btoa(t.join(""))
                .replace(/\//g,'_')
                .replace(/\+/g,'-')
                .replace(/=/g, '')
        },
        e.decodeBase64 = function(e) {
            if ("undefined" == typeof atob)
                return new Uint8Array(Array.prototype.slice.call(new Buffer(e,"base64"), 0));
            var n,
                t = atob(e.replace(/-/g,'+')
                    .replace(/_/g, '/')),
                r = new Uint8Array(t.length);
            for (n = 0; n < t.length; n++)
                r[n] = t.charCodeAt(n);
            return r
        }, e
});
