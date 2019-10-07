$(document).ready(function () {
    var $form = $('#form');
    var $input = $('#url');
    var $pageSum = $('#summary');
    var $error = $('#error');


    $form.submit(function (e) {

        $.ajax({
                url: url
            })
            .done(function (data) {
                var summary = "";
                var images = "";
                var title = "";
                var desc = "";

                for (each in data) {
                    switch (each) {
                        case 'title':
                            title = '<h3>' + data[each] + '</h3>';
                            break;
                        case 'description':
                            desc = '<p>' + data[each] + '</p>';
                            break;
                        case 'images':
                            data[each].forEach(function (img) {
                                if (!img['url']) {
                                    return;
                                }
                                images += '<img src="' + img['url'] + '"' + '>';
                            });
                            break;
                        default:
                            break;
                    }
                }
                summary = title + desc + images;
                $pageSum.html(summary);
            })
            .fail(function (error) {
                $error.text(error.responseText);
            });
    });

});