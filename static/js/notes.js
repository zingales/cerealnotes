"use strict";

var USERS_BY_ID = {};

const NOTE_TYPES = [
  'Marginalia',
  'Meta',
  'Prediction',
  'Question',
];

const classNamesByName = {
  noteTypeButton: 'note-type-button',
  primaryButton: 'mui-btn--primary',
  noteAuthorSpan: 'note-author',
  noteCategorySpan: 'note-category',
  noteTimeSpan: 'note-time',
  noteIdSpan: 'note-id',
  noteContent: 'note-content',

};

const classesByName = {
  noteTypeButton: '.' + classNamesByName.noteTypeButton,
};

// CREATE ELEMENTS
const $createButtonWithText = function(text) {
  return $('<button>').addClass('mui-btn').text(text);
};

const activateButton = function($button) {
  $button.addClass(classNamesByName.primaryButton);
};

const deactivateButton = function($button) {
  $button.removeClass(classNamesByName.primaryButton);
};

const isButtonActive = function($button) {
  return $button.hasClass(classNamesByName.primaryButton);
};

const $createHalfRowDivWithElement = function($element) {
  return $('<div>').addClass('mui-col-xs-6').append($element);
};

const $createRowWithTwoElements = function($element1, $element2) {
  const $row = $('<div>').addClass('mui-row');

  const $column1 = $createHalfRowDivWithElement($element1);
  const $column2 = $createHalfRowDivWithElement($element2);

  return $row
    .append($column1)
    .append($column2);
};

const $createGridContainer = function() {
  return $('<div>').addClass('mui-container-fluid');
};

const $createTextAreaDiv = function(labelText) {
  const $textarea = $('<textarea>').prop('required', true).prop('rows', 4);
  const $label = $('<label>').text(labelText);

  return $('<div>').addClass('mui-textfield')
    .append($textarea)
    .append($label);
};


// NOTES
function capitalizeFirstLetter(string) {
  return string.charAt(0).toUpperCase() + string.slice(1);
};

function assignCategory(noteId, $cateogry) {
  let id = `${noteId}_category`;
  $cateogry.prop('id', id);

  return $.get('/api/note-category?id=' + noteId, function(responseObj) {
    $(`#${id}`).text(capitalizeFirstLetter(responseObj.category));
  });
}

function $createNote(noteId, note) {
  let $newNote = $("#templates .note").clone();
  $newNote.find(`.${classNamesByName.noteIdSpan}`).text(noteId);
  $newNote.find(`.${classNamesByName.noteAuthorSpan}`).text(USERS_BY_ID[note.authorId].displayName);
  $newNote.find(`.${classNamesByName.noteTimeSpan}`).text(moment(note.creationTime).fromNow());
  $newNote.find(`.${classNamesByName.noteContent}`).text(note.content);

  // Assign type info
  assignCategory(noteId, $newNote.find(`.${classNamesByName.noteCategorySpan}`));

  return $newNote;

}

// ADD
function $createAddNoteModal() {
  const $modal = $('<div>').addClass('modal').addClass('mui-container')

  const $buttons = NOTE_TYPES.map(noteType => {
    return $createButtonWithText(noteType).addClass(classNamesByName.noteTypeButton);
  });

  const $textareaDiv = $createTextAreaDiv('Note').addClass('note-content');

  const $submitNoteButton = $createButtonWithText('Submit').addClass(classNamesByName.primaryButton);

  const onSubmitNoteClick = function() {
    const content = $textareaDiv.children()[0].value;
    let category = "";
    for (const $button of $buttons) {
      if ($button.hasClass(classNamesByName.primaryButton)) {
        category = $button.text();
        break;
      }
    }

    sendNewNote(content, category).then((_) => {
      mui.overlay('off');
      $textareaDiv.children()[0].value = "";
      for (const $button of $buttons) {
        if (isButtonActive($button)) {
          deactivateButton($button);
        }
      }
    }).finally(() => {
      refreshNotes();
    })

  }

  $submitNoteButton.click(onSubmitNoteClick);

  return $modal
    .append($createGridContainer()
      .append($createRowWithTwoElements($buttons[0], $buttons[1]))
      .append($createRowWithTwoElements($buttons[2], $buttons[3])))
    .append($textareaDiv)
    .append($submitNoteButton);
};

async function refreshNotes() {
  location.reload();
}

async function sendNewNote(noteContent, cateogry) {
  var data = await $.ajax({
    url: '/api/note',
    type: "POST",
    data: JSON.stringify({
      'content': noteContent
    }),
    contentType: "application/json; charset=utf-8",
    dataType: "json",
  }).fail(function(jqXHR, textStatus, errorThrown) {
    console.log("in note post")
    console.log(errorThrown);
  });

  if (cateogry) {
    const noteId = data.noteId;
    var bob = await $.ajax({
      url: '/api/note-category?id=' + noteId,
      type: "POST",
      data: JSON.stringify({
        'category': cateogry.toLowerCase()
      }),
      contentType: "application/json; charset=utf-8",
    }).fail(function(jqXHR, textStatus, errorThrown) {
      console.log("in note category post")
      console.log(errorThrown);
    });
  }
}

const activateModal = function($modal) {
  mui.overlay('on', $modal.getUnderlyingDomElement());
};

$(function() {
  const $addNoteModal = $createAddNoteModal();

  $.get('/api/user', function(usersById) {
    USERS_BY_ID = usersById;

    $.get('/api/note', function(notes) {
      const $notes = $('#notes');

      for (const key of Object.keys(notes)) {
        $notes.append($createNote(key, notes[key]));
      }
    });
  });

  $('#add-note-button').click(function() {
    activateModal($addNoteModal);
  });

  $(document).on('click', classesByName.noteTypeButton, function() {
    const $clickedButton = $(this);

    if (isButtonActive($clickedButton)) {
      deactivateButton($clickedButton);
    } else {
      $(classesByName.noteTypeButton).each(function() {
        const $button = $(this);

        if ($button.is($clickedButton)) {
          activateButton($button);
        } else {
          deactivateButton($button);
        }
      });
    }
  });
});
