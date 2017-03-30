AppConfig.guest = false;

var app = angular.module('souris-app', ['souris-common']);

app.config(['$routeProvider', function(route) {
  route
      .when('/',           HOME)
      .otherwise({redirectTo:'/'});
}]);

function menu_items(self, active) {
    return {items: [{link: '/',      title: "Home", active: "home" == active}]};
}

var HOME = {
    templateUrl: 'home.html',
    controller: ['$scope', '$remoteService', function(self, req) {}]
};
